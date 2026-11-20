#!/bin/bash
# Diagnóstico por etapa sin imprimir argumentos, configuración ni secretos.
aidex_stage() {
    AIDEX_STAGE=$1
    printf '[AIDEX %s] etapa=%s\n' "$AIDEX_SCRIPT" "$AIDEX_STAGE"
}

aidex_error() {
    printf '[AIDEX %s] ERROR etapa=%s archivo=%s linea=%s codigo=%s. Revisa /var/log/install.log.\n' \
        "$AIDEX_SCRIPT" "$AIDEX_STAGE" "$3" "$2" "$1" >&2
}

aidex_fail() {
    printf '[AIDEX %s] ERROR etapa=%s: %s\n' "$AIDEX_SCRIPT" "$AIDEX_STAGE" "$1" >&2
    exit 1
}

AIDEX_SCRIPT=${0##*/}
AIDEX_STAGE=inicio
trap 'aidex_error "$?" "$LINENO" "${BASH_SOURCE[0]##*/}"' ERR

# dscl presenta valores con espacios en varias líneas. Leer el plist evita
# depender de ese formato visual y conserva el valor exacto del atributo.
aidex_directory_value() {
    /usr/bin/dscl -plist . -read "$1" "$2" |
        /usr/bin/plutil -extract "dsAttrTypeStandard:$2.0" raw -o - -
}

aidex_has_record() {
    local records
    records=$(/usr/bin/dscl . -list "$1") || aidex_fail 'No se pudo consultar el directorio local de cuentas.'
    /usr/bin/grep -Fxq "$2" <<< "$records"
}

# Sustituir el arreglo completo: plutil sobre ProgramArguments.3 puede insertar
# un elemento y conservar el anterior. Esta operación también repara plists
# existentes con argumentos duplicados y es idempotente al reinstalar.
aidex_rebuild_program_arguments() {
    local executable configuration arguments
    executable=$(printf '%s' "$BINARY" | /usr/bin/sed -e 's/\&/\&amp;/g' -e 's/</\&lt;/g' -e 's/>/\&gt;/g')
    configuration=$(printf '%s' "$CONFIG" | /usr/bin/sed -e 's/\&/\&amp;/g' -e 's/</\&lt;/g' -e 's/>/\&gt;/g')
    arguments=$(printf '<array><string>%s</string><string>serve</string><string>--config</string><string>%s</string></array>' "$executable" "$configuration")
    /usr/bin/plutil -replace ProgramArguments -xml "$arguments" "$PLIST"
}

aidex_service_registered() {
    local status
    # Bash de macOS hereda ERR en la sustitución de comandos. La consulta tiene
    # manejo explícito de su código: ESRCH (113) no es un error de instalación.
    # Desactivar el trap solo en esta consulta evita el falso ERROR del subshell.
    if AIDEX_SERVICE_DETAILS=$(trap - ERR; /bin/launchctl print "system/$LABEL" 2>/dev/null); then
        return 0
    else
        status=$?
    fi
    [ "$status" -eq 113 ] && return 1
    aidex_fail "No se pudo consultar el servicio (launchctl print, código $status); no se continuará con la instalación."
}

aidex_stop_service() {
    local old_pid attempt status
    if aidex_service_registered; then
        old_pid=$(/usr/bin/awk '/^[[:space:]]*pid = [0-9]+/ {print $3; exit}' <<< "$AIDEX_SERVICE_DETAILS")
        if /bin/launchctl bootout "system/$LABEL"; then
            printf '[AIDEX %s] servicio=retirado; verificando terminación\n' "$AIDEX_SCRIPT"
        else
            status=$?
            # Puede desaparecer entre print y bootout. Aceptar solo ESRCH y
            # comprobar nuevamente la ausencia; los demás errores son fatales.
            [ "$status" -eq 113 ] || aidex_fail "No se pudo retirar el servicio (bootout, código $status)."
            if aidex_service_registered; then
                aidex_fail 'bootout devolvió 113 pero el servicio sigue registrado; no es seguro continuar.'
            fi
        fi
        # No migrar ni sustituir el ejecutable mientras el proceso anterior siga
        # activo. Espera acotada; no se fuerza la terminación de otros procesos.
        for ((attempt=0; attempt<60; attempt++)); do
            if ! aidex_service_registered; then
                if [ -z "$old_pid" ] || ! kill -0 "$old_pid" 2>/dev/null; then
                    printf '[AIDEX %s] servicio=detenido; resultado=correcto\n' "$AIDEX_SCRIPT"
                    return 0
                fi
            fi
            /bin/sleep 1
        done
        aidex_fail 'El servicio anterior no terminó en 60 segundos; se detuvo la actualización para proteger los datos.'
    else
        printf '[AIDEX %s] servicio=ausente; código=113; no requiere detenerse\n' "$AIDEX_SCRIPT"
    fi
}

aidex_read_address() {
    ADDRESS=$(/usr/bin/plutil -extract listen_address raw -o - "$CONFIG")
    PORT=${ADDRESS##*:}
    aidex_valid_port "$PORT" || aidex_fail 'El puerto configurado no es válido.'
}

aidex_valid_port() {
    case "$1" in ''|*[!0-9]*) return 1;; esac
    [ "${#1}" -le 5 ] && [ "$1" -ge 1 ] && [ "$1" -le 65535 ]
}

aidex_health_url() {
    local host scheme cert
    host=${ADDRESS%:*}
    case "$host" in 0.0.0.0) host=127.0.0.1;; '[::]') host='[::1]';; esac
    scheme=http
    if cert=$(/usr/bin/plutil -extract tls_certificate raw -o - "$CONFIG" 2>/dev/null) && [ -n "$cert" ]; then
        scheme=https
    fi
    printf '%s://%s:%s\n' "$scheme" "$host" "$PORT"
}

aidex_wait_service() {
    local attempt health
    health=$(aidex_health_url)
    for ((attempt=0; attempt<30; attempt++)); do
        if /usr/bin/curl --noproxy '*' -kfsS --connect-timeout 1 --max-time 2 "$health/health/live" 2>/dev/null |
            /usr/bin/grep -q '"product":"AIDEX"'; then
            return 0
        fi
        /bin/sleep 1
    done
    return 1
}

aidex_start_service() {
    aidex_stage verificar-puerto
    aidex_read_address
    if ! aidex_service_registered && /usr/sbin/lsof -nP -iTCP:"$PORT" -sTCP:LISTEN >/dev/null 2>&1; then
        aidex_fail "El puerto $PORT está ocupado por otro proceso. Elige otro puerto para AIDEX."
    fi
    aidex_stage habilitar-servicio
    /bin/launchctl enable "system/$LABEL"
    # Installation must load the repaired plist, even if launchd loaded the
    # previous definition when it was enabled.
    if [ "${1-}" = replace ]; then aidex_stop_service; fi
    aidex_stage registrar-servicio
    if aidex_service_registered; then
        /bin/launchctl kickstart "system/$LABEL"
    else
        /bin/launchctl bootstrap system "$PLIST"
    fi
    aidex_stage verificar-respuesta-http
    aidex_wait_service || aidex_fail "AIDEX no respondió en el puerto $PORT. Consulta Diagnóstico o el registro del servicio."
    if [ ! -f "$BASE/data/service-enabled" ]; then
        /usr/bin/sudo -u "$ACCOUNT" /usr/bin/touch "$BASE/data/service-enabled"
    fi
    echo "Servicio AIDEX operativo en el puerto $PORT."
}

aidex_change_port() {
    local old_port was_loaded=0 new_port=$1
    aidex_read_address
    old_port=$PORT
    aidex_valid_port "$new_port" || aidex_fail 'Introduce un puerto entre 1 y 65535.'
    if [ "$new_port" -eq "$old_port" ]; then return 0; fi
    if aidex_service_registered; then
        was_loaded=1
        aidex_stop_service
    fi
    if ! /usr/bin/sudo -u "$ACCOUNT" "$BINARY" configure-port --config "$CONFIG" --port "$new_port"; then
        if [ "$was_loaded" -eq 1 ]; then
            /bin/launchctl bootstrap system "$PLIST"
            aidex_wait_service || aidex_fail 'AIDEX no respondió en el puerto configurado. Consulta Diagnóstico.'
        fi
        return 1
    fi
    if [ "$was_loaded" -eq 1 ]; then
        PORT=$new_port
        if ! /bin/launchctl bootstrap system "$PLIST" || ! aidex_wait_service; then
            aidex_stop_service
            /usr/bin/sudo -u "$ACCOUNT" "$BINARY" configure-port --config "$CONFIG" --port "$old_port"
            /bin/launchctl bootstrap system "$PLIST"
            PORT=$old_port
            aidex_wait_service || aidex_fail 'Se restauró la configuración, pero AIDEX no respondió. Consulta Diagnóstico.'
            aidex_fail 'AIDEX no respondió en el nuevo puerto. Se restauró el puerto anterior.'
        fi
    fi
    echo "Puerto de AIDEX configurado en $new_port."
}
