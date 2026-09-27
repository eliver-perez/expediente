# Icono AIBID

`aibid-icon.svg` extrae únicamente las carpetas y la lupa de
`web/public/assets/brand/logo-v.svg`; los originales no cambian.
`aibid.ico` contiene 16, 24, 32, 40, 48, 64, 128 y 256 px con transparencia.
Los recursos COFF `cmd/*/resource_windows_amd64.syso` se incluyen automáticamente
por Go solo en Windows amd64. NSIS usa el mismo ICO para instalar/desinstalar.

Regeneración (Chrome, Node con dependencias web, ImageMagick y Go):

```
node scripts/build_windows_icon.mjs
```

Herramienta de recurso fijada: github.com/akavel/rsrc v0.10.2, MIT.
No se incorpora como dependencia de ejecución ni cambia go.mod.
Guía Microsoft: https://learn.microsoft.com/en-us/windows/win32/uxguide/vis-icons
Fuente de rsrc: https://github.com/akavel/rsrc
