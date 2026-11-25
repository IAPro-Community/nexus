# Uso y cuota

[🇬🇧 English](usage-and-quota.md) | [🇧🇷 Português](usage-and-quota.pt-BR.md) | **🇪🇸 Español**

`nexus usage` muestra snapshots de capacidad por proveedor y perfil. Cada snapshot indica origen, antigüedad del caché, límite conocido, consumo, saldo y ventana de renovación cuando el proveedor ofrece esos datos.

`UNKNOWN` significa que no existe un valor fiable; no significa cuota ilimitada. Use `--json` para automatización: sus claves y enums no cambian con el idioma. El selector evita cuentas en cooldown. `nexus config fallback enable` autoriza el cambio antes del inicio cuando un perfil está ocupado. Una vez iniciado el proveedor, Nexus no repite operaciones tras un límite de cuota/tasa si la salida o los efectos son ambiguos.
