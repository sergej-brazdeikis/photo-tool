package extended

// Layer identifies which matrix tier a row belongs to.
type Layer string

const (
	LayerFunctional Layer = "functional"
	LayerStepUX     Layer = "step_ux"
	LayerFlowUX     Layer = "flow_ux"
)

// UxAppMode records whether UX evidence came from the real binary or software driver.
type UxAppMode string

const (
	UxAppRealBinary      UxAppMode = "real_binary"
	UxAppSoftwareDriver  UxAppMode = "software_driver"
	UxAppNotApplicable   UxAppMode = ""
)

// Status is the row outcome in a matrix run.
type Status string

const (
	StatusPending Status = "pending"
	StatusPass    Status = "pass"
	StatusFail    Status = "fail"
	StatusSkip    Status = "skip"
	StatusManual  Status = "manual"
	StatusGap     Status = "gap"
)

// Row is one cell in the extended testing matrix.
type Row struct {
	ID            string    `json:"id" yaml:"id"`
	Story         string    `json:"story,omitempty" yaml:"story,omitempty"`
	Flow          string    `json:"flow,omitempty" yaml:"flow,omitempty"`
	Step          string    `json:"step,omitempty" yaml:"step,omitempty"`
	Layer         Layer     `json:"layer" yaml:"layer"`
	FR            []string  `json:"fr,omitempty" yaml:"fr,omitempty"`
	UxAppMode     UxAppMode `json:"ux_app_mode,omitempty" yaml:"ux_app_mode,omitempty"`
	Command       string    `json:"command,omitempty" yaml:"command,omitempty"`
	PNG           string    `json:"png,omitempty" yaml:"png,omitempty"`
	CaptureTool   string    `json:"capture_tool,omitempty" yaml:"capture_tool,omitempty"`
	JudgeSpec     string    `json:"judge_spec,omitempty" yaml:"judge_spec,omitempty"`
	JudgeScope    string    `json:"judge_scope,omitempty" yaml:"judge_scope,omitempty"`
	ParallelGroup string    `json:"parallel_group,omitempty" yaml:"parallel_group,omitempty"`
	Automatable   bool      `json:"automatable" yaml:"automatable"`
	Status        Status    `json:"status" yaml:"status"`
	Evidence      []string  `json:"evidence,omitempty" yaml:"evidence,omitempty"`
	Message       string    `json:"message,omitempty" yaml:"message,omitempty"`
	Steps         []string  `json:"steps,omitempty" yaml:"steps,omitempty"`
}

// Matrix is the full generated/run artifact.
type Matrix struct {
	GeneratedAt string `json:"generated_at"`
	GitShort    string `json:"git_short,omitempty"`
	Rows        []Row  `json:"rows"`
}
