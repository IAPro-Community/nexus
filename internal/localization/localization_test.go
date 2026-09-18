package localization

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestNormalize(t *testing.T) {
	cases := map[string]string{"pt": "pt-BR", "pt_PT.UTF-8": "pt-BR", "es-MX": "es", "en_US": "en", "de-DE": "en"}
	for input, want := range cases {
		if got := Normalize(input); got != want {
			t.Errorf("Normalize(%q)=%q want %q", input, got, want)
		}
	}
}

func TestResolvePrecedence(t *testing.T) {
	t.Setenv("AI_CLI_LANG", "es")
	if got := Resolve("pt-BR", "en"); got != "pt-BR" {
		t.Fatal(got)
	}
	if got := Resolve("", "en"); got != "es" {
		t.Fatal(got)
	}
	os.Unsetenv("AI_CLI_LANG")
}

func TestExtractGlobalFlagDoesNotConsumeProviderArgs(t *testing.T) {
	lang, rest, err := ExtractGlobalFlag([]string{"--lang", "es", "codex", "--lang", "provider-value"})
	if err != nil || lang != "es" || !reflect.DeepEqual(rest, []string{"codex", "--lang", "provider-value"}) {
		t.Fatalf("%q %#v %v", lang, rest, err)
	}
}

func TestCatalogParity(t *testing.T) {
	var reference map[string]string
	for _, name := range []string{"active.en.json", "active.pt-BR.json", "active.es.json"} {
		data, err := catalogFS.ReadFile("locales/" + name)
		if err != nil {
			t.Fatal(err)
		}
		var catalog map[string]string
		if err := json.Unmarshal(data, &catalog); err != nil {
			t.Fatal(err)
		}
		if reference == nil {
			reference = catalog
			continue
		}
		if len(catalog) != len(reference) {
			t.Fatalf("%s has %d keys; want %d", name, len(catalog), len(reference))
		}
		for key := range reference {
			if _, ok := catalog[key]; !ok {
				t.Errorf("%s missing %s", name, key)
			}
		}
	}
}

func TestHumanizeHelpKeepsMaestroUpdateExplicit(t *testing.T) {
	for _, lang := range []string{"en", "pt-BR", "es"} {
		Set(lang)
		got := HumanizeHelp("update — Update Nexus and Orquestrador Maestro to latest")
		if strings.Contains(got, "Nexus e Maestro") || strings.Contains(got, "Nexus y Maestro") || strings.Contains(got, "Nexus and Orquestrador Maestro") {
			t.Fatalf("%s retained merged update claim: %q", lang, got)
		}
	}
	Set(DefaultLanguage)
}
