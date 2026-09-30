# Selección de cuentas

[🇬🇧 English](account-selection.md) | [🇧🇷 Português](account-selection.pt-BR.md) | **🇪🇸 Español**

Nexus elige una cuenta utilizable dentro del proveedor solicitado. La decisión considera autenticación, cooldown, cuota conocida, prioridad, vinculación del proyecto y estrategia configurada. `UNKNOWN` sigue sin evidencia suficiente y no autoriza selección automática. Cuando el proveedor declara cuota `UNSUPPORTED`, la cuenta autenticada y saludable sigue siendo candidata como capacidad opaca, sin porcentaje inventado y por debajo de recursos con capacidad medida.

Orden práctico: perfil explícito, vínculo del workspace, perfil predeterminado y selección automática. Las cuentas deshabilitadas, no autenticadas o limitadas se rechazan. Si la cuenta elegida alcanza un límite, el fallback puede probar otra cuenta elegible del mismo proveedor.

Use `nexus explain <provider>` para ver la puntuación, `nexus usage` para capacidad y `nexus bind <provider>:<profile>` para una preferencia por proyecto. Los IDs y valores de `--json` permanecen estables.
