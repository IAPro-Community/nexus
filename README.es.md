# IAPro Nexus

**Una estación de trabajo local para operar coding agents.**

Proyectos, agentes persistentes, terminales reales, proveedores, worktrees y automatización en una experiencia integrada de Web, Desktop y CLI.

<p align="center"><a href="README.md">Português</a> · <a href="README.en.md">English</a> · <strong>Español</strong></p>

![Espacio de trabajo de IAPro Nexus](docs/assets/screenshots/workspace-overview.png)

## TL;DR

IAPro Nexus es una estación de trabajo local para trabajar con coding agents. Organiza proyectos, agentes persistentes, sesiones, terminales reales, proveedores, worktrees y uso/cuota alrededor de un Core común. El camino más sencillo es: abrir un proyecto, crear una AI Session, elegir un proveedor y trabajar en la terminal. Composer, Flow, Mission y Maestro son capas progresivas, no requisitos para el trabajo directo.

## Empieza en minutos

```bash
git clone https://github.com/kivervinicius/ai-cli.git
cd ai-cli
bun --cwd web install --frozen-lockfile
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
- [Tour visual](docs/product/visual-tour.md)
- [Flujo directo](docs/product/direct.md)
- [Composer](docs/product/composer.md)
- [Flow](docs/product/flow.md)
- [Desktop](docs/desktop/overview.md)
- [Arquitectura](docs/architecture/overview.md)
- [Troubleshooting](docs/operations/troubleshooting.md)

La [integración con Maestro](docs/product/maestro.md) es opcional para el camino Direct. Las afirmaciones sobre soporte, uso y continuidad dependen de evidencias; las capacidades planeadas o experimentales están marcadas.

## Comunidad

[Seguridad](SECURITY.md) · [Soporte](SUPPORT.md) · [Gobernanza](GOVERNANCE.md) · [Roadmap](ROADMAP.md) · [Changelog](CHANGELOG.md) · [Licencia MIT](LICENSE)
