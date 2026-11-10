package contacts

import (
	"testing"

	"outreach/internal/fields"
)

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

func TestParseCSVDoesNotGuessWorkEmail(t *testing.T) {
	t.Parallel()
	p, err := ParseCSV("Work Email,First\nada@acme.com,Ada\n", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if p.Guessed["email"] != "" {
		t.Fatalf("guessed %+v", p.Guessed)
	}
	if len(p.Rows) != 0 {
		t.Fatalf("rows %+v", p.Rows)
	}
	if len(p.Headers) != 2 || p.Headers[0] != "Work Email" {
		t.Fatalf("headers %v", p.Headers)
	}
	if len(p.Samples) != 1 || p.Samples[0][0] != "ada@acme.com" {
		t.Fatalf("samples %v", p.Samples)
	}
}

func TestParseCSVMapsWorkEmailWhenTold(t *testing.T) {
	t.Parallel()
	p, err := ParseCSV("Work Email,First\nada@acme.com,Ada\n", ColumnMap{
		"email":      "Work Email",
		"first_name": "First",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Rows) != 1 || p.Rows[0].Email != "ada@acme.com" || p.Rows[0].FirstName != "Ada" {
		t.Fatalf("%+v", p.Rows)
	}
}

func TestParseCSVGuessesCustomFieldByLabel(t *testing.T) {
	t.Parallel()
	p, err := ParseCSV("email,Seniority\nada@acme.com,Staff\n", nil, []fields.Hint{{Key: "seniority", Label: "Seniority"}})
	if err != nil {
		t.Fatal(err)
	}
	if p.Guessed["seniority"] != "Seniority" {
		t.Fatalf("guessed %+v", p.Guessed)
	}
	if p.Rows[0].Custom["seniority"] != "Staff" {
		t.Fatalf("%+v", p.Rows[0])
	}
}
