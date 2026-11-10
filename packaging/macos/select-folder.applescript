on run
    display dialog "Abre AIDEX y pulsa Seleccionar carpeta." with title "Seleccionar carpeta AIDEX" buttons {"Aceptar"} default button "Aceptar"
end run
on open location requestURL
    try
        with timeout of 150 seconds
            do shell script quoted form of "/Library/Application Support/AIDEX/bin/aidex-folder-picker" & " " & quoted form of requestURL
        end timeout
    on error detailText number errorNumber
        if errorNumber is not -128 then
            display dialog "No se pudo seleccionar la carpeta. Vuelve a AIDEX para intentarlo de nuevo o introduce la ruta manualmente." with title "Seleccionar carpeta AIDEX" buttons {"Aceptar"} default button "Aceptar"
        end if
    end try
end open location
