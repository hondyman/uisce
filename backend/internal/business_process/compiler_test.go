package business_process

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCompileProcessTemplate(t *testing.T) {
	tests := []struct {
		name    string
		template ProcessTemplate
		wantErr  bool
	}{
		{
			name: "Valid Template",
			template: ProcessTemplate{
				ProcessID: "test_process",
				Steps: []Step{
					{ID: "start", Name: "Start"},
					{ID: "end", Name: "End"},
				},
				Transitions: []Transition{
					{From: "start", To: "end"},
				},
			},
			wantErr: false,
		},
		{
			name: "Missing ProcessID",
			template: ProcessTemplate{
				Steps: []Step{{ID: "start"}},
			},
			wantErr: true,
		},
		{
			name: "Duplicate Step ID",
			template: ProcessTemplate{
				ProcessID: "dup_step",
				Steps: []Step{
					{ID: "start"},
					{ID: "start"},
				},
			},
			wantErr: true,
		},
		{
			name: "Invalid Transition From",
			template: ProcessTemplate{
				ProcessID: "invalid_trans",
				Steps: []Step{
					{ID: "start"},
				},
				Transitions: []Transition{
					{From: "unknown", To: "start"},
				},
			},
			wantErr: true,
		},
		{
			name: "Invalid Transition To",
			template: ProcessTemplate{
				ProcessID: "invalid_trans_to",
				Steps: []Step{
					{ID: "start"},
				},
				Transitions: []Transition{
					{From: "start", To: "unknown"},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CompileProcessTemplate(tt.template)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetNextStep(t *testing.T) {
	tmpl := ProcessTemplate{
		Transitions: []Transition{
			{From: "step1", To: "step2"},
		},
	}

	next, err := GetNextStep(tmpl, "step1")
	assert.NoError(t, err)
	assert.Equal(t, "step2", next)

	next, err = GetNextStep(tmpl, "step2")
	assert.NoError(t, err)
	assert.Equal(t, "", next)
}
