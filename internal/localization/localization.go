package localization

import (
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"

	systemlocale "github.com/Xuanwo/go-locale"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

const DefaultLanguage = "en"

var Supported = []string{"pt-BR", "en", "es"}

//go:embed locales/*.json
var catalogFS embed.FS

var (
	bundle  *i18n.Bundle
	mu      sync.RWMutex
	current = DefaultLanguage
)

func init() {
	bundle = i18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("json", json.Unmarshal)
	for _, name := range []string{"active.en.json", "active.pt-BR.json", "active.es.json"} {
		if _, err := bundle.LoadMessageFileFS(catalogFS, "locales/"+name); err != nil {
			panic(err)
		}
	}
}

func Normalize(value string) string {
	tag := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(strings.SplitN(value, ".", 2)[0]), "_", "-"))
	switch {
	case tag == "pt" || strings.HasPrefix(tag, "pt-"):
		return "pt-BR"
	case tag == "es" || strings.HasPrefix(tag, "es-"):
		return "es"
	case tag == "en" || strings.HasPrefix(tag, "en-"):
		return "en"
	default:
		return DefaultLanguage
	}
}

func IsSupported(value string) bool {
	if value == "auto" {
		return true
	}
	for _, candidate := range Supported {
		if value == candidate {
			return true
		}
	}
	return false
}

func isRecognized(value string) bool {
	tag := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(value), "_", "-"))
	return tag == "auto" || tag == "pt" || strings.HasPrefix(tag, "pt-") || tag == "es" || strings.HasPrefix(tag, "es-") || tag == "en" || strings.HasPrefix(tag, "en-")
}

func Resolve(flagValue, configured string) string {
	if flagValue != "" && flagValue != "auto" {
		return Normalize(flagValue)
	}
	if value := os.Getenv("AI_CLI_LANG"); value != "" {
		return Normalize(value)
	}
	if configured != "" && configured != "auto" {
		return Normalize(configured)
	}
	if tag, err := systemlocale.Detect(); err == nil {
		return Normalize(tag.String())
	}
	for _, key := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		if value := os.Getenv(key); value != "" {
			return Normalize(value)
		}
	}
	return DefaultLanguage
}

func Set(value string) { mu.Lock(); current = Normalize(value); mu.Unlock() }
func Current() string  { mu.RLock(); defer mu.RUnlock(); return current }

func T(id string, data ...map[string]any) string {
	template := map[string]any(nil)
	if len(data) > 0 {
		template = data[0]
	}
	localizer := i18n.NewLocalizer(bundle, Current(), DefaultLanguage)
	value, err := localizer.Localize(&i18n.LocalizeConfig{MessageID: id, TemplateData: template})
	if err != nil {
		return id
	}
	return value
}

// HumanizeHelp localizes human descriptions while preserving command names,
// flags, provider IDs, and spacing used by terminal help output.
func HumanizeHelp(value string) string {
	translations := map[string]map[string]string{
		"pt-BR": {
			"Open interactive Workspace OS (TUI)": "Abrir o Workspace OS interativo (TUI)", "Launch IAPro Nexus Workspace OS (Web UI)": "Abrir o IAPro Nexus Workspace OS (Web)", "Nexus Supervised Agent Runtimes": "Execuções supervisionadas de Agentes", "Start supervised agent runtime & attach": "Iniciar execução supervisionada e conectar", "Stop running supervised runtime": "Parar execução supervisionada", "List active running supervised runtimes": "Listar execuções supervisionadas ativas", "Attach terminal to running runtime": "Conectar terminal a uma execução", "Same-provider account handoff": "Transferir conta no mesmo provedor", "Cross-provider context handoff": "Transferir contexto entre provedores", "Launch provider with intelligent account selection": "Abrir provedor com seleção inteligente", "Launch specific profile": "Abrir perfil específico", "Explicit auto-selection": "Seleção automática explícita", "Resume previous session using provider-native syntax": "Retomar sessão com sintaxe nativa", "List installed providers, versions & capabilities": "Listar provedores, versões e capacidades", "List configured profiles, auth status & priorities": "Listar perfis, autenticação e prioridades", "Add a new provider authentication profile": "Adicionar perfil de autenticação", "Remove an existing profile": "Remover perfil existente", "Run provider official login/logout flow": "Executar login/logout oficial", "Set default active profile for provider": "Definir perfil padrão do provedor", "Display profile health, plan and account status": "Exibir saúde, plano e status da conta", "Display real-time quota metrics & cache freshness": "Exibir quota e atualidade do cache", "Inspect profile configuration details": "Inspecionar configuração do perfil", "Universal session index across all providers": "Índice universal de sessões", "View workspaces, session history & bindings": "Ver espaços, histórico e vínculos", "Bind current workspace to a preferred profile": "Vincular espaço ao perfil preferido", "Unbind current workspace": "Remover vínculo do espaço", "List all active workspace bindings": "Listar vínculos ativos", "Explain account selection decision and scores": "Explicar seleção e pontuações", "Deep diagnostics of runtime, keyrings & CLIs": "Diagnóstico de execuções, chaveiros e CLIs", "Audit file sharing and isolation boundary": "Auditar compartilhamento e isolamento", "View local session execution log": "Ver log local de sessões", "Aggregated statistics (sessions, fallbacks, rate limits)": "Estatísticas agregadas", "Manage control plane settings": "Gerenciar configurações", "Update Nexus and Orquestrador Maestro to latest": "Atualizar Nexus e Maestro", "Generate shell completion scripts": "Gerar scripts de autocomplete", "Display build and platform information": "Exibir informações de build e plataforma",
		},
		"es": {
			"Open interactive Workspace OS (TUI)": "Abrir Workspace OS interativo (TUI)", "Launch IAPro Nexus Workspace OS (Web UI)": "Abrir IAPro Nexus Workspace OS (Web)", "Nexus Supervised Agent Runtimes": "Runtimes supervisados de Agentes", "Start supervised agent runtime & attach": "Iniciar runtime supervisado y conectar", "Stop running supervised runtime": "Detener runtime supervisado", "List active running supervised runtimes": "Listar runtimes supervisados activos", "Attach terminal to running runtime": "Conectar terminal al runtime", "Same-provider account handoff": "Transferir cuenta en el mismo proveedor", "Cross-provider context handoff": "Transferir contexto entre proveedores", "Launch provider with intelligent account selection": "Abrir proveedor con selección inteligente", "Launch specific profile": "Abrir perfil específico", "Explicit auto-selection": "Selección automática explícita", "Resume previous session using provider-native syntax": "Reanudar sesión con sintaxis nativa", "List installed providers, versions & capabilities": "Listar proveedores, versions y capacidades", "List configured profiles, auth status & priorities": "Listar perfiles, autenticación y prioridades", "Add a new provider authentication profile": "Agregar perfil de autenticación", "Remove an existing profile": "Eliminar perfil existente", "Run provider official login/logout flow": "Ejecutar login/logout oficial", "Set default active profile for provider": "Definir perfil predeterminado", "Display profile health, plan and account status": "Mostrar salud, plan y estado de cuenta", "Display real-time quota metrics & cache freshness": "Mostrar cuota y vigencia del caché", "Inspect profile configuration details": "Inspeccionar configuración del perfil", "Universal session index across all providers": "Índice universal de sesiones", "View workspaces, session history & bindings": "Ver espacios, historial y vínculos", "Bind current workspace to a preferred profile": "Vincular espacio al perfil preferido", "Unbind current workspace": "Eliminar vínculo del espacio", "List all active workspace bindings": "Listar vínculos activos", "Explain account selection decision and scores": "Explicar selección y puntuaciones", "Deep diagnostics of runtime, keyrings & CLIs": "Diagnóstico de runtimes, llaveros y CLI", "Audit file sharing and isolation boundary": "Auditar archivos y aislamiento", "View local session execution log": "Ver log local de sesiones", "Aggregated statistics (sessions, fallbacks, rate limits)": "Estadísticas agregadas", "Manage control plane settings": "Gestionar configuración", "Update Nexus and Orquestrador Maestro to latest": "Actualizar Nexus y Maestro", "Generate shell completion scripts": "Generar scripts de autocompletado", "Display build and platform information": "Mostrar información de build y plataforma",
		},
	}
	for source, target := range translations[Current()] {
		value = strings.ReplaceAll(value, source, target)
	}
	// Keep legacy merged-help text truthful after localization. Nexus updates
	// and Maestro maintenance are separate explicit operations.
	value = strings.ReplaceAll(value, "Update Nexus and Orquestrador Maestro to latest", "Update Nexus only (Maestro is separate)")
	value = strings.ReplaceAll(value, "Atualizar Nexus e Maestro", "Atualizar Nexus (Maestro é separado)")
	value = strings.ReplaceAll(value, "Actualizar Nexus y Maestro", "Actualizar Nexus (Maestro es independiente)")
	return value
}

func ExtractGlobalFlag(args []string) (string, []string, error) {
	if len(args) == 0 {
		return "", args, nil
	}
	if args[0] == "--lang" {
		if len(args) < 2 {
			return "", nil, fmt.Errorf("%s", T("error.lang_value"))
		}
		if !isRecognized(args[1]) {
			return "", nil, fmt.Errorf(T("error.unsupported_language"), args[1])
		}
		return args[1], args[2:], nil
	}
	if strings.HasPrefix(args[0], "--lang=") {
		value := strings.TrimPrefix(args[0], "--lang=")
		if value == "" {
			return "", nil, fmt.Errorf("%s", T("error.lang_value"))
		}
		if !isRecognized(value) {
			return "", nil, fmt.Errorf(T("error.unsupported_language"), value)
		}
		return value, args[1:], nil
	}
	return "", args, nil
}
