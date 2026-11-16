package ai

import (
	"errors"
	"testing"
)

func TestRegistryOmitsFake(t *testing.T) {
	t.Parallel()
	for _, p := range Registry() {
		if p.Kind == KindFake || p.Kind == "" || p.Label == "" {
			t.Fatalf("%+v", p)
		}
		if p.Embed && len(p.EmbedModels) == 0 && !p.CustomModels {
			t.Fatalf("embed list empty %+v", p)
		}
		if !p.CustomModels && len(p.Compose) == 0 {
			t.Fatalf("compose list empty %+v", p)
		}
	}
	if _, ok := Lookup(KindFake); ok {
		t.Fatal("fake in catalog")
	}
	if _, ok := Lookup(KindOpenAI); !ok {
		t.Fatal("openai missing")
	}
}

func TestCheckModelUsesCatalog(t *testing.T) {
	t.Parallel()
	spec, _ := Lookup(KindOpenAI)
	if err := checkModel(spec, RoleCompose, "gpt-6-luna"); err != nil {
		t.Fatal(err)
	}
	if err := checkModel(spec, RoleCompose, "not-a-model"); !errors.Is(err, ErrUnknownModel) {
		t.Fatalf("%v", err)
	}
	compat, _ := Lookup(KindCompat)
	if err := checkModel(compat, RoleCompose, "local-llama"); err != nil {
		t.Fatal(err)
	}
	router, ok := Lookup(KindOpenRouter)
	if !ok || !router.CustomModels || !router.Embed || router.NeedsBaseURL {
		t.Fatalf("%+v", router)
	}
}
