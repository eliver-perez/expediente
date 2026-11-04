on run
    set aviso to "Desinstalar AIDEX retira la aplicación y el servicio. Por defecto conserva la base de datos, usuarios, configuración, índices, historial y licencia local para una reinstalación." & return & return & "La desinstalación no libera la activación en el servidor. Para trasladar la licencia, desactívala antes desde AIDEX. Borrar los datos locales tampoco desactiva la licencia." & return & return & "Los documentos originales, incluidas bibliotecas administradas y cargas, siempre se conservan."
    set eleccion to button returned of (display dialog aviso with title "Desinstalar AIDEX" buttons {"Cancelar", "Eliminar datos", "Conservar datos"} default button "Conservar datos")
    if eleccion is "Cancelar" then return
    set accion to "conservar"
    if eleccion is "Eliminar datos" then
        set advertencia to "Se perderán los usuarios, bibliotecas registradas, índices, historial, cachés, registros, configuración y estado local de licencia. Los originales externos permanecerán en sus rutas. Los documentos y archivos no reconocidos dentro de los datos internos se conservarán aparte en AIDEX → Documentos conservados; se indicará la ubicación al terminar." & return & return & "Esto NO libera la activación en el servidor. Escribe ELIMINAR para confirmar."
        set confirmacion to display dialog advertencia with title "Confirmar eliminación de datos" default answer "" buttons {"Cancelar", "Continuar"} default button "Cancelar" with icon caution
        if button returned of confirmacion is "Cancelar" then return
        if text returned of confirmacion is not "ELIMINAR" then
            display dialog "No se eliminaron datos: la confirmación no coincide." with title "Desinstalar AIDEX" buttons {"Aceptar"} default button "Aceptar"
            return
        end if
        set accion to "eliminar"
    end if
    try
        -- Ejecutar siempre el auxiliar instalado por pkg como root, no una copia
        -- de la app que pudiera estar en una carpeta modificable por el usuario.
        set helperPath to "/Applications/AIDEX/Desinstalar AIDEX.app/Contents/Resources/uninstall-helper.sh"
        set resultado to do shell script (quoted form of helperPath & " " & accion) with administrator privileges
        display dialog resultado with title "Resultado de desinstalación" buttons {"Aceptar"} default button "Aceptar" with icon note
    on error detalle number codigo
        if codigo is -128 then return
        display dialog "No se completó la desinstalación." & return & return & detalle with title "Error al desinstalar AIDEX" buttons {"Aceptar"} default button "Aceptar" with icon stop
    end try
end run
