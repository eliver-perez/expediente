import AppKit

@main
final class ControlCenter: NSObject, NSApplicationDelegate {
    private let title = "Centro de Control de AIDEX"
    private let service = NSTextField(labelWithString: "No disponible")
    private let database = NSTextField(labelWithString: "No disponible")
    private let port = NSTextField(labelWithString: "Puerto actual: no disponible")
    private let feedback = NSTextField(wrappingLabelWithString: "")
    private let start = NSButton(title: "Iniciar", target: nil, action: nil)
    private let stop = NSButton(title: "Detener", target: nil, action: nil)
    private let restart = NSButton(title: "Reiniciar", target: nil, action: nil)
    private let changePort = NSButton(title: "Cambiar puerto…", target: nil, action: nil)
    private let diagnose = NSButton(title: "Diagnóstico…", target: nil, action: nil)
    private let browser = NSButton(title: "Abrir AIDEX", target: nil, action: nil)
    private let setup = NSButton(title: "Crear administrador inicial…", target: nil, action: nil)
    private var window: NSWindow!
    private var status = ControlStatus()
    private var refreshing = false
    private var busy = false
    private var timer: Timer?

    static func main() {
        let app = NSApplication.shared
        let delegate = ControlCenter()
        app.setActivationPolicy(.regular)
        app.delegate = delegate
        app.run()
    }

    func applicationDidFinishLaunching(_ notification: Notification) {
        let menu = NSMenu()
        let appItem = NSMenuItem()
        menu.addItem(appItem)
        appItem.submenu = NSMenu()
        appItem.submenu?.addItem(withTitle: "Salir del Centro de Control", action: #selector(NSApplication.terminate(_:)), keyEquivalent: "q")
        NSApp.mainMenu = menu

        window = NSWindow(contentRect: NSRect(x: 0, y: 0, width: 560, height: 365),
                          styleMask: [.titled, .closable, .miniaturizable], backing: .buffered, defer: false)
        window.title = title
        window.isReleasedWhenClosed = false
        let content = window.contentView!
        let header = NSView()
        header.wantsLayer = true
        header.layer?.backgroundColor = NSColor(srgbRed: 19/255, green: 46/255, blue: 56/255, alpha: 1).cgColor
        let heading = NSTextField(labelWithString: title)
        heading.textColor = .white
        heading.font = .systemFont(ofSize: 24, weight: .semibold)
        let badge = NSImageView(image: NSApp.applicationIconImage)
        badge.imageScaling = .scaleProportionallyUpOrDown
        let branding = NSStackView(views: [badge, heading])
        branding.spacing = 14
        branding.translatesAutoresizingMaskIntoConstraints = false
        header.addSubview(branding)

        port.font = .systemFont(ofSize: NSFont.systemFontSize, weight: .semibold)
        feedback.textColor = .secondaryLabelColor
        feedback.maximumNumberOfLines = 3
        feedback.preferredMaxLayoutWidth = 506
        let divider = NSBox()
        divider.boxType = .separator
        let body = NSStackView(views: [
            statusRow("Servicio AIDEX", service),
            row([start, stop, restart]),
            statusRow("Servidor de base de datos", database),
            row([port, changePort]),
            divider,
            row([browser, diagnose, setup]),
            feedback
        ])
        body.orientation = .vertical
        body.alignment = .leading
        body.spacing = 15
        body.translatesAutoresizingMaskIntoConstraints = false
        header.translatesAutoresizingMaskIntoConstraints = false
        content.addSubview(header)
        content.addSubview(body)
        NSLayoutConstraint.activate([
            header.topAnchor.constraint(equalTo: content.topAnchor),
            header.leadingAnchor.constraint(equalTo: content.leadingAnchor),
            header.trailingAnchor.constraint(equalTo: content.trailingAnchor),
            header.heightAnchor.constraint(equalToConstant: 78),
            branding.leadingAnchor.constraint(equalTo: header.leadingAnchor, constant: 24),
            branding.centerYAnchor.constraint(equalTo: header.centerYAnchor),
            badge.widthAnchor.constraint(equalToConstant: 42),
            badge.heightAnchor.constraint(equalToConstant: 42),
            body.topAnchor.constraint(equalTo: header.bottomAnchor, constant: 24),
            body.leadingAnchor.constraint(equalTo: content.leadingAnchor, constant: 26),
            body.trailingAnchor.constraint(equalTo: content.trailingAnchor, constant: -26),
            body.bottomAnchor.constraint(lessThanOrEqualTo: content.bottomAnchor, constant: -23),
            divider.widthAnchor.constraint(equalTo: body.widthAnchor)
        ])
        for (button, action) in [
            (start, #selector(startService)), (stop, #selector(stopService)),
            (restart, #selector(restartService)), (changePort, #selector(editPort)),
            (browser, #selector(openBrowser)), (diagnose, #selector(showDiagnostics)),
            (setup, #selector(createAdministrator))
        ] {
            button.target = self
            button.action = action
            button.controlSize = .large
            button.isEnabled = false
        }
        feedback.stringValue = "Consultando el estado de AIDEX…"
        window.center()
        window.makeKeyAndOrderFront(nil)
        NSApp.activate(ignoringOtherApps: true)
        refresh()
        timer = Timer.scheduledTimer(withTimeInterval: 4, repeats: true) { [weak self] _ in self?.refresh() }
    }

    func applicationShouldTerminateAfterLastWindowClosed(_ sender: NSApplication) -> Bool { true }

    private func row(_ views: [NSView]) -> NSStackView {
        let stack = NSStackView(views: views)
        stack.alignment = .centerY
        stack.spacing = 10
        return stack
    }

    private func statusRow(_ title: String, _ value: NSTextField) -> NSStackView {
        let label = NSTextField(labelWithString: title)
        label.font = .systemFont(ofSize: NSFont.systemFontSize, weight: .semibold)
        label.widthAnchor.constraint(equalToConstant: 187).isActive = true
        return row([label, value])
    }

    private func applyStatus() {
        service.stringValue = status.service.rawValue
        database.stringValue = status.database.rawValue
        service.textColor = status.service == .running ? .systemGreen : .secondaryLabelColor
        database.textColor = status.database == .running ? .systemGreen : .secondaryLabelColor
        port.stringValue = "Puerto actual: " + (status.endpoint.map { String($0.port) } ?? "no disponible")
        let active = status.service == .running || status.service == .starting
        start.isEnabled = !busy && status.service == .stopped && status.setup == "configured"
        stop.isEnabled = !busy && active
        restart.isEnabled = !busy && active
        changePort.isEnabled = !busy && status.endpoint != nil && status.service != .unavailable
        browser.isEnabled = !busy && status.service == .running && status.endpoint != nil
        diagnose.isEnabled = !busy
        setup.isEnabled = !busy && status.service == .stopped && status.setup == "required"
    }

    private func refresh() {
        guard !busy, !refreshing else { return }
        refreshing = true
        DispatchQueue.global(qos: .userInitiated).async {
            let latest = ControlService.status()
            DispatchQueue.main.async {
                self.refreshing = false
                guard !self.busy else { return }
                self.status = latest
                if self.feedback.stringValue == "Consultando el estado de AIDEX…" {
                    self.feedback.stringValue = latest.setup == "required" ? "Crea el administrador inicial para empezar a usar AIDEX." : ""
                }
                self.applyStatus()
            }
        }
    }

    private func perform(_ operation: String, progress: String, port: Int? = nil, form: AdministratorForm? = nil) {
        busy = true
        feedback.stringValue = progress
        applyStatus()
        DispatchQueue.global(qos: .userInitiated).async {
            let result = Result { try form.map(ControlService.setup) ?? ControlService.run(operation, port: port) }
            let latest = ControlService.status()
            DispatchQueue.main.async {
                self.busy = false
                self.status = latest
                self.feedback.stringValue = ""
                self.applyStatus()
                switch result {
                case .success(let response) where response.cancelled:
                    break
                case .success(let response) where operation == "doctor":
                    self.presentDiagnostics(response)
                case .success(let response) where response.code == 0:
                    if form != nil { self.notice("Administrador inicial creado", "Ya puedes abrir AIDEX e iniciar sesión.") }
                case .success(let response):
                    self.presentError(response.output, operation: operation)
                case .failure:
                    self.notice("No se pudo completar la operación", "Abre el Centro de Control desde Aplicaciones y vuelve a intentarlo.", warning: true)
                }
            }
        }
    }

    @objc private func startService() { perform("start", progress: "Iniciando AIDEX…") }
    @objc private func stopService() { perform("stop", progress: "Deteniendo AIDEX…") }
    @objc private func restartService() { perform("restart", progress: "Reiniciando AIDEX…") }
    @objc private func showDiagnostics() { perform("doctor", progress: "Comprobando la configuración y las herramientas locales…") }
    @objc private func openBrowser() {
        if let url = status.endpoint?.url { NSWorkspace.shared.open(url) }
    }

    @objc private func editPort() {
        guard let current = status.endpoint?.port else { return }
        let alert = NSAlert()
        alert.messageText = "Cambiar puerto de AIDEX"
        alert.informativeText = "Puerto actual: \(current). Introduce un número entre 1 y 65535."
        alert.addButton(withTitle: "Guardar")
        alert.addButton(withTitle: "Cancelar")
        let input = NSTextField(string: String(current))
        input.frame = NSRect(x: 0, y: 0, width: 240, height: 24)
        input.setAccessibilityLabel("Nuevo puerto")
        alert.accessoryView = input
        alert.window.initialFirstResponder = input
        while alert.runModal() == .alertFirstButtonReturn {
            if let port = Int(input.stringValue.trimmingCharacters(in: .whitespaces)), (1...65535).contains(port) {
                if port != current { perform("port", progress: "Cambiando el puerto de AIDEX…", port: port) }
                return
            }
            alert.informativeText = "Introduce un puerto válido entre 1 y 65535."
        }
    }

    @objc private func createAdministrator() {
        let alert = NSAlert()
        alert.messageText = "Crear administrador inicial"
        alert.informativeText = "Usa esta cuenta para iniciar sesión en AIDEX. La contraseña debe tener entre 6 y 128 caracteres."
        alert.addButton(withTitle: "Crear administrador")
        alert.addButton(withTitle: "Cancelar")
        let username = NSTextField()
        let name = NSTextField()
        let password = NSSecureTextField()
        let confirmation = NSSecureTextField()
        defer { password.stringValue = ""; confirmation.stringValue = "" }
        let fields = zip(["Usuario", "Nombre visible", "Contraseña", "Repite la contraseña"], [username, name, password, confirmation])
        let grid = NSGridView(views: fields.map { label, input in
            input.setAccessibilityLabel(label)
            input.widthAnchor.constraint(equalToConstant: 225).isActive = true
            return [NSTextField(labelWithString: label), input]
        })
        grid.rowSpacing = 8
        grid.columnSpacing = 16
        grid.column(at: 0).xPlacement = .trailing
        grid.frame.size = grid.fittingSize
        alert.accessoryView = grid
        alert.window.initialFirstResponder = username
        while alert.runModal() == .alertFirstButtonReturn {
            let user = username.stringValue.trimmingCharacters(in: .whitespacesAndNewlines)
            let display = name.stringValue.trimmingCharacters(in: .whitespacesAndNewlines)
            if user.isEmpty || display.isEmpty || !(6...128).contains(password.stringValue.unicodeScalars.count) {
                alert.informativeText = "Completa el usuario, el nombre visible y una contraseña de 6 a 128 caracteres."
            } else if password.stringValue != confirmation.stringValue {
                alert.informativeText = "Las contraseñas no coinciden."
            } else {
                let form = AdministratorForm(username: user, displayName: display, password: password.stringValue,
                                             confirmation: confirmation.stringValue)
                perform("setup", progress: "Creando el administrador inicial…", form: form)
                return
            }
        }
    }

    private func presentDiagnostics(_ result: CommandResult) {
        var details = [
            "Servicio AIDEX: \(status.service.rawValue)",
            "Servidor de base de datos: \(status.database.rawValue)",
            port.stringValue
        ]
        if result.output.contains("Configuración: OK") {
            details.append("Configuración de AIDEX: correcta")
            details.append(result.code == 0 ? "Herramientas PDF y OCR en español e inglés: disponibles"
                           : "No se pudieron verificar las herramientas PDF y OCR. Revisa Poppler y Tesseract con los idiomas español e inglés.")
        } else {
            details.append("No se pudo leer la configuración de AIDEX.")
        }
        if status.setup == "required" { details.append("El administrador inicial está pendiente.") }
        notice("Diagnóstico de AIDEX", details.joined(separator: "\n\n"))
    }

    private func presentError(_ output: String, operation: String) {
        let message: String
        if output.contains("Administrador inicial creado.") {
            message = "La cuenta se creó. AIDEX sigue detenido o no respondió. Revisa el puerto y el diagnóstico antes de iniciarlo."
        } else if output.contains("puerto anterior") {
            message = "AIDEX no respondió en el nuevo puerto. Se restauró el puerto anterior."
        } else if output.contains("ocupado") {
            message = "El puerto está ocupado o no está disponible. Elige otro e inténtalo de nuevo."
        } else if output.contains("HTTPS personalizada") {
            message = "Esta instalación usa una configuración de red personalizada. Ajusta su puerto mediante la administración avanzada."
        } else if output.contains("ya fue creado") || output.contains("ya existen usuarios") {
            message = "El administrador inicial ya existe. Puedes iniciar AIDEX desde el Centro de Control."
        } else if output.contains("falta crear el primer administrador") {
            message = "Crea el administrador inicial antes de iniciar AIDEX."
        } else if operation == "setup" && (output.contains("contraseña") || output.contains("usuario") || output.contains("nombre")) {
            message = "Revisa los datos de la cuenta. El usuario admite letras inglesas, números, punto, guion, arroba y guion bajo."
        } else {
            message = "Revisa el estado del servicio en Diagnóstico y vuelve a intentarlo."
        }
        notice("No se pudo completar la operación", message, warning: true)
    }

    private func notice(_ title: String, _ message: String, warning: Bool = false) {
        let alert = NSAlert()
        alert.messageText = title
        alert.informativeText = message
        alert.alertStyle = warning ? .warning : .informational
        alert.addButton(withTitle: "Aceptar")
        alert.runModal()
    }
}
