package contacts

import "testing"

func TestPreviewCSVMapsHeadersAndFlagsDupes(t *testing.T) {
	t.Parallel()
	rows, err := PreviewCSV("email,first_name,company\nada@acme.com,Ada,Acme\ncareers@acme.com,HR,Acme\nada@acme.com,Ada,Acme\n")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 {
		t.Fatalf("len %d", len(rows))
	}
	if rows[0].Email != "ada@acme.com" || rows[0].FirstName != "Ada" || rows[0].Company != "Acme" {
		t.Fatalf("%+v", rows[0])
	}
	if !rows[1].Generic || rows[1].RoleClass != "generic_mailbox" || rows[1].Readiness != "risky" {
		t.Fatalf("%+v", rows[1])
	}
	if !rows[0].Duplicate || !rows[2].Duplicate {
		t.Fatalf("dupes %+v %+v", rows[0], rows[2])
	}
}

func TestPreviewCSVRejectsEmpty(t *testing.T) {
	t.Parallel()
	if _, err := PreviewCSV("  "); err != ErrCSV {
		t.Fatalf("%v", err)
	}
}

func TestPreviewCSVRequiresEmail(t *testing.T) {
	t.Parallel()
	rows, err := PreviewCSV("email,first_name\n,Ada")
	if err != nil || len(rows) != 1 || rows[0].Readiness != "blocked" {
		t.Fatalf("%+v %v", rows, err)
	}
}
