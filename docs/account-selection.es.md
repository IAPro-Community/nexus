# Selección de cuentas

[🇬🇧 English](account-selection.md) | [🇧🇷 Português](account-selection.pt-BR.md) | **🇪🇸 Español**

Nexus elige una cuenta utilizable dentro del proveedor solicitado. La decisión considera autenticación, cooldown, cuota conocida, prioridad, vinculación del proyecto y estrategia configurada. La selección automática requiere cuota reciente `LIVE` o `CACHED` confiable. `UNKNOWN`, `ERROR`, `ESTIMATED` y `UNSUPPORTED` no autorizan una selección automática ni reciben capacidad inventada. Un perfil fijado, vínculo o predeterminado explícito sigue la política de selección explícita sin clasificarse como cuota medida.

Orden práctico: perfil explícito, vínculo del workspace, perfil predeterminado y selección automática. Las cuentas deshabilitadas, no autenticadas, en cooldown, agotadas o sin cuota confiable reciente se rechazan de la selección automática. El cambio automático entre cuentas está deshabilitado por defecto; actívelo explícitamente con `nexus config fallback enable` y desactívelo con `nexus config fallback disable`. La opción permite cambiar antes del inicio cuando un perfil está ocupado, sujeto a identidad y cuota reciente. Una vez iniciado el proceso del proveedor, Nexus no repite una operación por límite de cuota/tasa porque la salida y los efectos externos pueden ser ambiguos.

En Nexus, `nexus <provider>:balance` y `nexus <provider>:auto` rotan los nuevos inicios en orden round-robin entre perfiles elegibles con cuota reciente `LIVE`/`CACHED` confiable. El cursor se guarda por proveedor y se serializa entre procesos de Nexus. Se omiten cuentas deshabilitadas, no autenticadas, en cooldown, agotadas o sin evidencia confiable. El comando simple `nexus <provider>` mantiene la clasificación por capacidad verificada, y `nexus <provider>:<profile>` sigue fijado al perfil indicado.

Use `nexus explain <provider>` para ver la puntuación, `nexus usage` para capacidad y `nexus bind <provider>:<profile>` para una preferencia por proyecto. Los IDs y valores de `--json` permanecen estables.
