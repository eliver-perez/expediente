on run
    display dialog "Abre AIDEX en esta Mac, selecciona una carpeta y pulsa Autorizar carpeta." with title "Autorizar carpeta AIDEX" buttons {"Aceptar"} default button "Aceptar"
end run

on open location requestURL
    set helper to "/Library/Application Support/AIDEX/bin/aidex-folder-helper"
    try
        -- El auxiliar valida el esquema, puerto, ticket y alcance. La ruta no
        -- forma parte de una orden shell; solo se transfieren datos entre comillas.
        set descriptionText to do shell script quoted form of helper & " describe " & quoted form of requestURL
        set fingerprint to paragraph 1 of descriptionText
        set detailText to text ((length of fingerprint) + 2) thru -1 of descriptionText
        display dialog detailText with title "Autorizar carpeta AIDEX" buttons {"Cancelar", "Autorizar carpeta"} default button "Cancelar" with icon caution
        with timeout of 300 seconds
            set resultText to do shell script quoted form of helper & " apply " & quoted form of requestURL & " " & quoted form of fingerprint with administrator privileges
        end timeout
        display dialog resultText with title "Acceso verificado por AIDEX" buttons {"Aceptar"} default button "Aceptar"
    on error detailText number errorNumber
        if errorNumber is not -128 then
            display dialog "No se completó la autorización." & return & return & detailText with title "Autorizar carpeta AIDEX" buttons {"Aceptar"} default button "Aceptar" with icon caution
        end if
    end try
end open location
