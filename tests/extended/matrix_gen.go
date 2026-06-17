package extended

import (
	_ "embed"
	"fmt"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

//go:embed definitions.yaml
var definitionsYAML []byte

type definitionsFile struct {
	FunctionalGroups []struct {
		ID            string `yaml:"id"`
		ParallelGroup string `yaml:"parallel_group"`
		Command       string `yaml:"command"`
	} `yaml:"functional_groups"`
	StoryFunctional []struct {
		Story         string   `yaml:"story"`
		Flow          string   `yaml:"flow"`
		FR            []string `yaml:"fr"`
		Command       string   `yaml:"command"`
		ParallelGroup string   `yaml:"parallel_group"`
	} `yaml:"story_functional"`
	NFRFunctional []struct {
		ID            string   `yaml:"id"`
		FR            []string `yaml:"fr"`
		Command       string   `yaml:"command"`
		ParallelGroup string   `yaml:"parallel_group"`
	} `yaml:"nfr_functional"`
	StepUX []struct {
		Step  string `yaml:"step"`
		Story string `yaml:"story"`
		Flow  string `yaml:"flow"`
		PNG   string `yaml:"png"`
	} `yaml:"step_ux"`
	FlowUX []struct {
		Flow  string   `yaml:"flow"`
		Steps []string `yaml:"steps"`
	} `yaml:"flow_ux"`
	ManualRows []struct {
		ID      string   `yaml:"id"`
		Story   string   `yaml:"story"`
		Flow    string   `yaml:"flow"`
		Layer   string   `yaml:"layer"`
		FR      []string `yaml:"fr"`
		Message string   `yaml:"message"`
	} `yaml:"manual_rows"`
	ScaleUnit []struct {
		ID            string   `yaml:"id"`
		Tier          string   `yaml:"tier"`
		FR            []string `yaml:"fr"`
		Command       string   `yaml:"command"`
		ParallelGroup string   `yaml:"parallel_group"`
	} `yaml:"scale_unit"`
	ScaleFunctional []struct {
		ID            string   `yaml:"id"`
		Tier          string   `yaml:"tier"`
		FR            []string `yaml:"fr"`
		Command       string   `yaml:"command"`
		ParallelGroup string   `yaml:"parallel_group"`
	} `yaml:"scale_functional"`
	UxScaleSpot []struct {
		Step  string `yaml:"step"`
		Story string `yaml:"story"`
		Flow  string `yaml:"flow"`
		PNG   string `yaml:"png"`
	} `yaml:"ux_scale_spot"`
	UxEdge []struct {
		Step  string `yaml:"step"`
		Story string `yaml:"story"`
		Flow  string `yaml:"flow"`
		PNG   string `yaml:"png"`
	} `yaml:"ux_edge"`
	UxLayout []struct {
		Step  string `yaml:"step"`
		Story string `yaml:"story"`
		Flow  string `yaml:"flow"`
		PNG   string `yaml:"png"`
	} `yaml:"ux_layout"`
}

// GenerateMatrix builds the full matrix from embedded definitions.
func GenerateMatrix(gitShort string) (*Matrix, error) {
	var def definitionsFile
	if err := yaml.Unmarshal(definitionsYAML, &def); err != nil {
		return nil, fmt.Errorf("parse definitions: %w", err)
	}
	m := &Matrix{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		GitShort:    gitShort,
	}
	seen := map[string]struct{}{}

	add := func(r Row) error {
		if r.ID == "" {
			return fmt.Errorf("row missing id")
		}
		if _, ok := seen[r.ID]; ok {
			return fmt.Errorf("duplicate row id %q", r.ID)
		}
		seen[r.ID] = struct{}{}
		if r.Status == "" {
			r.Status = StatusPending
		}
		if r.Automatable || r.Layer == LayerFunctional || r.Layer == LayerStepUX || r.Layer == LayerFlowUX {
			// default automatable true for test layers
		}
		m.Rows = append(m.Rows, r)
		return nil
	}

	for _, g := range def.FunctionalGroups {
		if err := add(Row{
			ID:            "functional-" + g.ID,
			Layer:         LayerFunctional,
			Flow:          strings.TrimPrefix(g.ID, "epic"),
			Command:       g.Command,
			ParallelGroup: g.ParallelGroup,
			Automatable:   true,
			Status:        StatusPending,
		}); err != nil {
			return nil, err
		}
	}

	for _, s := range def.StoryFunctional {
		if err := add(Row{
			ID:            fmt.Sprintf("story-%s-functional", s.Story),
			Story:         s.Story,
			Flow:          s.Flow,
			Layer:         LayerFunctional,
			FR:            s.FR,
			Command:       s.Command,
			ParallelGroup: s.ParallelGroup,
			Automatable:   true,
			Status:        StatusPending,
		}); err != nil {
			return nil, err
		}
	}

	for _, n := range def.NFRFunctional {
		if err := add(Row{
			ID:            n.ID,
			Layer:         LayerFunctional,
			FR:            n.FR,
			Command:       n.Command,
			ParallelGroup: n.ParallelGroup,
			Automatable:   true,
			Status:        StatusPending,
		}); err != nil {
			return nil, err
		}
	}

	const stepJudge = "_bmad-output/test-artifacts/judge-prompt-step-ux.md"
	for _, s := range def.StepUX {
		if err := add(Row{
			ID:          fmt.Sprintf("step-%s-ux", s.Step),
			Story:       s.Story,
			Flow:        s.Flow,
			Step:        s.Step,
			Layer:       LayerStepUX,
			UxAppMode:   UxAppRealBinary,
			PNG:         s.PNG,
			CaptureTool: "photo-tool-gui-journey",
			JudgeSpec:   stepJudge,
			JudgeScope:  "single_step",
			Automatable: true,
			Status:      StatusPending,
		}); err != nil {
			return nil, err
		}
	}

	const flowJudge = "_bmad-output/test-artifacts/judge-prompt-flow-ux.md"
	for _, f := range def.FlowUX {
		if err := add(Row{
			ID:          fmt.Sprintf("flow-%s-ux", f.Flow),
			Flow:        f.Flow,
			Layer:       LayerFlowUX,
			UxAppMode:   UxAppRealBinary,
			Steps:       f.Steps,
			JudgeSpec:   flowJudge,
			JudgeScope:  "flow_summary",
			Automatable: true,
			Status:      StatusPending,
		}); err != nil {
			return nil, err
		}
	}

	for _, m := range def.ManualRows {
		layer := Layer(m.Layer)
		if layer == "" {
			layer = LayerFunctional
		}
		if err := add(Row{
			ID:          m.ID,
			Story:       m.Story,
			Flow:        m.Flow,
			Layer:       layer,
			FR:          m.FR,
			Message:     m.Message,
			Automatable: false,
			Status:      StatusManual,
		}); err != nil {
			return nil, err
		}
	}

	const scaleJudge = "_bmad-output/test-artifacts/judge-prompt-scale-ux.md"
	for _, s := range def.ScaleUnit {
		if err := add(Row{
			ID:            "scale-unit-" + s.ID,
			Layer:         LayerScaleUnit,
			FR:            s.FR,
			Command:       s.Command,
			ParallelGroup: s.ParallelGroup,
			Message:       "tier=" + s.Tier + "; skipped with go test -short",
			Automatable:   true,
			Status:        StatusPending,
		}); err != nil {
			return nil, err
		}
	}
	for _, s := range def.ScaleFunctional {
		if err := add(Row{
			ID:            "scale-func-" + s.ID,
			Layer:         LayerScaleFunctional,
			FR:            s.FR,
			Command:       s.Command,
			ParallelGroup: s.ParallelGroup,
			Message:       "tier=" + s.Tier + "; skipped with go test -short",
			Automatable:   true,
			Status:        StatusPending,
		}); err != nil {
			return nil, err
		}
	}
	addScaleUX := func(layer Layer, items []struct {
		Step  string `yaml:"step"`
		Story string `yaml:"story"`
		Flow  string `yaml:"flow"`
		PNG   string `yaml:"png"`
	}) error {
		for _, s := range items {
			if err := add(Row{
				ID:          fmt.Sprintf("%s-%s", layer, s.Step),
				Story:       s.Story,
				Flow:        s.Flow,
				Step:        s.Step,
				Layer:       layer,
				UxAppMode:   UxAppRealBinary,
				PNG:         s.PNG,
				CaptureTool: "photo-tool-gui-journey",
				JudgeSpec:   scaleJudge,
				JudgeScope:  "single_step",
				Automatable: true,
				Status:      StatusPending,
			}); err != nil {
				return err
			}
		}
		return nil
	}
	if err := addScaleUX(LayerUxScaleSpot, def.UxScaleSpot); err != nil {
		return nil, err
	}
	if err := addScaleUX(LayerUxEdge, def.UxEdge); err != nil {
		return nil, err
	}
	if err := addScaleUX(LayerUxLayout, def.UxLayout); err != nil {
		return nil, err
	}

	return m, nil
}

// AllStoriesPresent reports missing story IDs from 1.1–4.1 in functional rows.
func AllStoriesPresent(m *Matrix) []string {
	want := []string{
		"1.1", "1.2", "1.3", "1.4", "1.5", "1.6", "1.7", "1.8",
		"2.1", "2.2", "2.3", "2.4", "2.5", "2.6", "2.7", "2.8", "2.9", "2.10", "2.11", "2.12",
		"3.1", "3.2", "3.3", "3.4", "3.5",
		"4.1",
	}
	got := map[string]struct{}{}
	for _, r := range m.Rows {
		if r.Layer == LayerFunctional && r.Story != "" {
			got[r.Story] = struct{}{}
		}
	}
	var missing []string
	for _, s := range want {
		if _, ok := got[s]; !ok {
			missing = append(missing, s)
		}
	}
	return missing
}
