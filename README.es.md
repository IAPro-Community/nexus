# IAPro Nexus

> **Estado comprobado (27/10/2026):** el source de `release/v1.0.0` declara `1.0.0-beta.11`. **v1 estable aún no está homologada ni publicada.** No suponga URLs de instaladores a partir de tags antiguas.

**Una estación de trabajo local para operar coding agents.**

Proyectos, agentes persistentes, sesiones de terminal supervisadas, proveedores, worktrees y automatización en una experiencia integrada de Web, Desktop y CLI.

<p align="center"><a href="README.md">Português</a> · <a href="README.en.md">English</a> · <strong>Español</strong></p>

![Espacio de trabajo de IAPro Nexus](docs/assets/screenshots/workspace-overview.png)

## TL;DR

IAPro Nexus es una estación de trabajo local para trabajar con coding agents. Organiza proyectos, agentes persistentes, sesiones, terminales supervisados, proveedores, worktrees y uso/cuota alrededor de un Core común. El camino más sencillo es: abrir un proyecto, crear una AI Session, elegir un proveedor y trabajar en la terminal. Composer, Flow, Mission y Maestro son capas progresivas, no requisitos para el trabajo directo.

## Empieza en minutos

El candidato actual `1.0.0-beta.11` integra plataformas en el código, pero el soporte público depende de pruebas nativas del SHA exacto. El repositorio de source ya es público; la auditoría de historial, licencias y secretos continúa pendiente. No se garantiza un instalador público sin verificar artefactos firmados reales.

### Desarrolladores (compilar desde el código)

```bash
git clone https://github.com/IAPro-Community/nexus.git
cd nexus
bun install --cwd web --frozen-lockfile
make build
./nexus doctor
./nexus web
```

Después sigue **+ New → AI Session → proveedor → terminal**. No necesitas abrir Composer, Flow o Mission para trabajar directamente.

## Web, Desktop y CLI

| Superficie | Mejor para |
| --- | --- |
| **Nexus Web** | Control, observación y administración local desde el navegador. |
| **Nexus Desktop** | Shell nativo Wails cuando existan evidencias de build y runtime para la plataforma. |
| **Nexus CLI** | Scripts, diagnóstico, operación headless y flujos de terminal. |

Consulta la [matriz de plataformas](docs/operations/platform-support.md) antes de considerar soportada una capacidad nativa.

## Explora

- [Mapa de documentación](docs/README.md)
- [Getting started (índice)](docs/getting-started/README.md)
- [Instalación nativa](docs/getting-started/installation.md)
- [Paquetes nativos](docs/getting-started/packages.md)
- [Docker (compatibilidad)](docs/getting-started/docker-compat.md)
- [Tour visual](docs/product/visual-tour.md)
- [Flujo directo](docs/product/direct.md)
- [Composer](docs/product/composer.md)
- [Flow](docs/product/flow.md)
- [Desktop](docs/desktop/overview.md)
- [Arquitectura](docs/architecture/overview.md)
- [Troubleshooting](docs/operations/troubleshooting.md)

La [integración con Maestro](docs/product/maestro.md) es opcional para el camino Direct. Las afirmaciones sobre soporte, uso y continuidad dependen de evidencias; las capacidades planeadas o experimentales están marcadas.

## Comunidad

[TL;DR](docs/TLDR.md) · [Documentación](docs/README.md) · [Seguridad](SECURITY.md) · [Soporte](SUPPORT.md) · [Gobernanza](GOVERNANCE.md) · [Roadmap](ROADMAP.md) · [Changelog](CHANGELOG.md) · [PolyForm Strict](LICENSE)

**Licencia vigente:** [PolyForm Strict 1.0.0](LICENSE), código visible pero **no open source OSI**. No concede permisos generales para modificar o redistribuir el proyecto. **Licencia objetivo:** Apache-2.0, **aún no vigente**. El cambio depende de verificar autoría, derechos de terceros/assets y seguridad del historial. Publicar el código no autoriza distribuir forks modificados.

**Objetivo comunitario:** Nexus Community completo, local, sin límites artificiales en Composer, Flow, Agents o Missions; financiación futura por donaciones, servicios opcionales y soporte empresarial. Los modelos/proveedores externos pueden tener costes propios.

[Plan de transición open source](docs/legal/OPEN_SOURCE_TRANSITION.md) · [Licenciamiento actual](docs/legal/LICENSING.md) · [Contribuciones](CONTRIBUTING.md) · [Forks y upstream](docs/community/FORKS_AND_UPSTREAM.md) · [Marca](docs/legal/TRADEMARK_POLICY.md) · [Financiación](docs/community/FUNDING.md) · [Avisos de terceros](docs/legal/THIRD_PARTY_NOTICES.md).

La versión estable sigue **NO-GO** hasta la validación del SHA exacto.
