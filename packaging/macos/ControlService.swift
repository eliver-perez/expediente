import Darwin
import Foundation

struct AdministratorForm: Encodable {
    let username: String
    let displayName: String
    let password: String
    let confirmation: String

    enum CodingKeys: String, CodingKey {
        case username, password, confirmation
        case displayName = "display_name"
    }
}

struct ControlStatus {
    enum Service: String {
        case running = "En ejecución"
        case starting = "Iniciando"
        case stopped = "Detenido"
        case unavailable = "No disponible"
    }

    struct Endpoint: Decodable {
        let port: Int
        let url: URL
    }
    struct Database: Decodable {
        let port: Int
        let setup: String
    }

    var service: Service = .unavailable
    var database: Service = .unavailable
    var endpoint: Endpoint?
    var setup: String?
}

struct CommandResult {
    let code: Int32
    let output: String

    var cancelled: Bool { output.contains("(-128)") }
}

private final class LocalCommand {
    let process = Process()
    let output = Pipe()

    init(_ executable: String, _ arguments: [String]) throws {
        process.executableURL = URL(fileURLWithPath: executable)
        process.arguments = arguments
        process.environment = ["PATH": "/usr/bin:/bin:/usr/sbin:/sbin", "LC_ALL": "en_US.UTF-8"]
        process.standardInput = FileHandle.nullDevice
        process.standardOutput = output
        process.standardError = output
        try process.run()
    }

    func result() -> CommandResult {
        let contents = output.fileHandleForReading.readDataToEndOfFile()
        process.waitUntilExit()
        return CommandResult(code: process.terminationStatus, output: String(decoding: contents, as: UTF8.self))
    }
}

enum ControlService {
    private static let base = "/Library/Application Support/AIDEX"
    private static let runtime = base + "/bin/tools/postgresql/bin"
    private static let helper = base + "/bin/aidex-manage"

    static func status() -> ControlStatus {
        var status = ControlStatus()
        guard FileManager.default.isExecutableFile(atPath: helper) else { return status }
        let decoder = JSONDecoder()
        if let contents = try? Data(contentsOf: URL(fileURLWithPath: base + "/control/endpoint.json")),
           let endpoint = try? decoder.decode(ControlStatus.Endpoint.self, from: contents),
           (1...65535).contains(endpoint.port), ["http", "https"].contains(endpoint.url.scheme ?? ""),
           endpoint.url.host != nil, endpoint.url.user == nil, endpoint.url.password == nil {
            status.endpoint = endpoint
        }
        if let result = try? LocalCommand("/bin/launchctl", ["print", "system/app.aidex"]).result() {
            if result.code == 113 {
                status.service = .stopped
            } else if result.code == 0 {
                let state = result.output.split(separator: "\n").map { $0.trimmingCharacters(in: .whitespaces) }
                if state.contains("state = running") {
                    status.service = .running
                } else if state.contains(where: { $0 == "state = spawn scheduled" || $0 == "state = spawning" || $0.hasPrefix("pid = ") }) {
                    status.service = .starting
                } else {
                    status.service = .stopped
                }
            }
        }
        if let contents = try? Data(contentsOf: URL(fileURLWithPath: base + "/control/database.json")),
           let database = try? decoder.decode(ControlStatus.Database.self, from: contents),
           (1024...65535).contains(database.port) {
            status.setup = database.setup
            if let result = try? LocalCommand(runtime + "/pg_isready", [
                "-q", "-h", "127.0.0.1", "-p", String(database.port), "-U", "aidex_app", "-d", "aidex", "-t", "1"
            ]).result() {
                switch result.code {
                case 0: status.database = .running
                case 1: status.database = .starting
                case 2: status.database = status.service == .stopped ? .stopped : .unavailable
                default: break
                }
            }
        }
        return status
    }

    private static func authorize(_ arguments: [String]) throws -> LocalCommand {
        let script = """
        on run argv
            set commandText to ""
            repeat with argumentValue in argv
                set commandText to commandText & " " & quoted form of (contents of argumentValue)
            end repeat
            return do shell script commandText with administrator privileges altering line endings false
        end run
        """
        return try LocalCommand("/usr/bin/osascript", ["-e", script, helper] + arguments)
    }

    static func run(_ operation: String, port: Int? = nil) throws -> CommandResult {
        let arguments = [operation] + (port.map { [String($0)] } ?? [])
        return try authorize(arguments).result()
    }

    static func setup(_ form: AdministratorForm) throws -> CommandResult {
        let directory = "/private/tmp/aidex-" + UUID().uuidString
        try FileManager.default.createDirectory(atPath: directory, withIntermediateDirectories: false,
                                                attributes: [.posixPermissions: 0o700])
        defer { try? FileManager.default.removeItem(atPath: directory) }
        let path = directory + "/setup.sock"
        let listener = Darwin.socket(AF_UNIX, SOCK_STREAM, 0)
        guard listener >= 0 else { throw posixError() }
        defer { Darwin.close(listener) }
        var socket = sockaddr_un()
        socket.sun_family = sa_family_t(AF_UNIX)
        let length = MemoryLayout<sockaddr_un>.size
        socket.sun_len = UInt8(length)
        withUnsafeMutableBytes(of: &socket.sun_path) { $0.copyBytes(from: path.utf8) }
        let bound = withUnsafePointer(to: &socket) {
            $0.withMemoryRebound(to: sockaddr.self, capacity: 1) {
                Darwin.bind(listener, $0, socklen_t(length))
            }
        }
        guard bound == 0, Darwin.listen(listener, 4) == 0 else { throw posixError() }
        let command = try authorize(["setup", path, String(geteuid())])
        while command.process.isRunning {
            var descriptor = pollfd(fd: listener, events: Int16(POLLIN), revents: 0)
            if Darwin.poll(&descriptor, 1, 250) <= 0 { continue }
            let client = Darwin.accept(listener, nil, nil)
            if client < 0 { continue }
            defer { Darwin.close(client) }
            var uid: uid_t = 0
            var gid: gid_t = 0
            guard getpeereid(client, &uid, &gid) == 0, uid == 0 else { continue }
            var noSignal: Int32 = 1
            guard setsockopt(client, SOL_SOCKET, SO_NOSIGPIPE, &noSignal, socklen_t(MemoryLayout<Int32>.size)) == 0 else {
                throw posixError()
            }
            var payload = try JSONEncoder().encode(form)
            defer { payload.resetBytes(in: 0..<payload.count) }
            try FileHandle(fileDescriptor: client, closeOnDealloc: false).write(contentsOf: payload)
            Darwin.shutdown(client, SHUT_WR)
            break
        }
        return command.result()
    }

    private static func posixError() -> POSIXError {
        POSIXError(POSIXErrorCode(rawValue: errno) ?? .EIO)
    }
}
