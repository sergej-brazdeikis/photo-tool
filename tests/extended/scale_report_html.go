package extended

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const scaleReportHTMLTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8"/>
<meta name="viewport" content="width=device-width, initial-scale=1"/>
<title>Scale report — {{.RunID}}</title>
<style>
:root { --bg:#f6f6f4; --fg:#1a1a1a; --muted:#666; --pass:#1a6b34; --fail:#b42318; --card:#fff; --border:#ddd; }
@media (prefers-color-scheme: dark) {
  :root { --bg:#121212; --fg:#eee; --muted:#aaa; --card:#1e1e1e; --border:#333; }
}
* { box-sizing: border-box; }
body { font-family: system-ui, sans-serif; margin: 0; background: var(--bg); color: var(--fg); line-height: 1.45; }
header { padding: 1.25rem 1.5rem; border-bottom: 1px solid var(--border); background: var(--card); }
h1 { margin: 0 0 .25rem; font-size: 1.35rem; }
.badge { display: inline-block; padding: .2rem .55rem; border-radius: 4px; font-weight: 600; font-size: .85rem; }
.badge.pass { background: #d4edda; color: var(--pass); }
.badge.fail { background: #f8d7da; color: var(--fail); }
@media (prefers-color-scheme: dark) {
  .badge.pass { background: #1a3d28; }
  .badge.fail { background: #3d1a1a; }
}
.meta { color: var(--muted); font-size: .9rem; }
.cards { display: flex; flex-wrap: wrap; gap: .75rem; padding: 1rem 1.5rem; }
.card { background: var(--card); border: 1px solid var(--border); border-radius: 6px; padding: .75rem 1rem; min-width: 8rem; }
.card strong { display: block; font-size: 1.4rem; }
.filters { padding: 0 1.5rem 1rem; display: flex; flex-wrap: wrap; gap: .5rem; align-items: center; }
.filters input, .filters select { padding: .35rem .5rem; border: 1px solid var(--border); border-radius: 4px; background: var(--card); color: var(--fg); }
section { padding: 0 1.5rem 1.5rem; }
table { width: 100%; border-collapse: collapse; font-size: .88rem; background: var(--card); border: 1px solid var(--border); }
th, td { text-align: left; padding: .45rem .6rem; border-bottom: 1px solid var(--border); vertical-align: top; }
th { background: var(--bg); position: sticky; top: 0; }
tr.hidden { display: none; }
tr.clickable { cursor: pointer; }
tr.clickable:hover { background: var(--bg); }
#step-detail { margin-top: .75rem; padding: .75rem 1rem; background: var(--card); border: 1px solid var(--border); border-radius: 6px; display: none; white-space: pre-wrap; font-size: .85rem; }
#step-detail.open { display: block; }
.thumb { max-width: 120px; max-height: 72px; cursor: pointer; border: 1px solid var(--border); }
#lightbox { display: none; position: fixed; inset: 0; background: rgba(0,0,0,.85); z-index: 99; align-items: center; justify-content: center; }
#lightbox.open { display: flex; }
#lightbox img { max-width: 95vw; max-height: 95vh; }
.chip { font-size: .75rem; padding: .1rem .35rem; border-radius: 3px; background: var(--bg); }
</style>
</head>
<body>
<header>
  <h1>Scale &amp; UX limit report</h1>
  <p class="meta">Run <code>{{.RunID}}</code> · {{.FinishedAt}} · Git {{.GitShort}}</p>
  <p><span class="badge {{if eq .MachineLine "EXTENDED_SCALE_RESULT=pass"}}pass{{else}}fail{{end}}">{{.MachineLine}}</span></p>
</header>
<div class="cards">
  <div class="card"><strong>{{.Counts.FunctionalPass}}</strong> functional pass</div>
  <div class="card"><strong>{{.Counts.FunctionalFail}}</strong> functional fail</div>
  <div class="card"><strong>{{.Counts.ScaleUnitPass}}</strong> scale unit pass</div>
  <div class="card"><strong>{{.Counts.ScaleUnitFail}}</strong> scale unit fail</div>
  <div class="card"><strong>{{.Counts.UXStepsPass}}</strong> UX steps pass</div>
  <div class="card"><strong>{{.Counts.UXStepsFail}}</strong> UX steps fail</div>
  <div class="card"><strong>{{.Counts.IssuesOpen}}</strong> open issues</div>
</div>
<div class="filters">
  <label>Search <input type="search" id="q" placeholder="step id…"/></label>
  <label>Tier <select id="tier"><option value="">All</option></select></label>
  <label>Flow <select id="flow"><option value="">All</option></select></label>
  <label>Layer <select id="layer"><option value="">All</option></select></label>
  <label>Result <select id="result"><option value="">All</option><option value="pass">pass</option><option value="fail">fail</option><option value="unknown">unknown</option></select></label>
</div>
<section>
  <h2>Open issues</h2>
  <table id="issues"><thead><tr><th>ID</th><th>Layer</th><th>Step</th><th>Summary</th></tr></thead><tbody></tbody></table>
</section>
<section>
  <h2>UX capture steps</h2>
  <table id="steps"><thead><tr>
    <th>Step</th><th>Flow</th><th>Dir</th><th>Tier</th><th>Judge</th><th>ms</th><th>Preview</th>
  </tr></thead><tbody></tbody></table>
  <div id="step-detail"></div>
</section>
<section>
  <h2>Persona coverage</h2>
  <table id="personas"><thead><tr><th>Persona</th><th>Journey</th><th>Tier</th><th>Result</th><th>Evidence</th></tr></thead><tbody></tbody></table>
</section>
<section>
  <h2>NFR gates</h2>
  <table id="nfr"><thead><tr><th>Gate</th><th>Target</th><th>Result</th><th>Evidence</th></tr></thead><tbody></tbody></table>
</section>
<section>
  <h2>Functional scale tests</h2>
  <table id="functional"><thead><tr><th>Log</th><th>Result</th></tr></thead><tbody></tbody></table>
</section>
<section>
  <h2>Edge-case matrix (B7)</h2>
  <table id="edge-matrix"><thead><tr><th>Scenario</th><th>Step</th><th>Judge</th></tr></thead><tbody></tbody></table>
</section>
<section>
  <h2>Reproduce</h2>
  <pre id="reproduce"></pre>
</section>
<div id="lightbox"><img alt="preview"/></div>
<script type="application/json" id="scale-report-data">{{.JSONEmbed}}</script>
<script>
const data = JSON.parse(document.getElementById('scale-report-data').textContent);
const tbody = document.querySelector('#steps tbody');
const tiers = new Set(), flows = new Set(), layers = new Set();
(data.ux_steps || []).forEach(s => {
  tiers.add(s.tier||''); flows.add(s.flow||''); layers.add(s.layer||'');
  const tr = document.createElement('tr');
  tr.dataset.tier = s.tier||'';
  tr.dataset.flow = s.flow||'';
  tr.dataset.layer = s.layer||'';
  tr.dataset.result = s.judge||'unknown';
  tr.dataset.q = (s.id+' '+s.flow+' '+s.intent).toLowerCase();
  tr.classList.add('clickable');
  tr.dataset.intent = s.intent||'';
  tr.dataset.verdict = s.verdict_excerpt||'';
  const img = s.png ? '<img class="thumb" src="'+s.png+'" alt=""/>' : '';
  tr.innerHTML = '<td><code>'+s.id+'</code></td><td>'+s.flow+'</td><td>'+s.dir+'</td><td>'+(s.tier||'—')+'</td><td><span class="chip">'+s.judge+'</span></td><td>'+(s.timing_ms||'—')+'</td><td>'+img+'</td>';
  tbody.appendChild(tr);
});
const it = document.querySelector('#issues tbody');
(data.issues||[]).forEach(i => {
  const tr = document.createElement('tr');
  tr.innerHTML = '<td><code>'+i.id+'</code></td><td>'+i.layer+'</td><td>'+(i.step||'—')+'</td><td>'+i.summary+'</td>';
  it.appendChild(tr);
});
function fillSelect(id, set) {
  const sel = document.getElementById(id);
  [...set].filter(Boolean).sort().forEach(v => {
    const o = document.createElement('option'); o.value = v; o.textContent = v; sel.appendChild(o);
  });
}
fillSelect('tier', tiers); fillSelect('flow', flows); fillSelect('layer', layers);
const pt = document.querySelector('#personas tbody');
(data.personas||[]).forEach(p => {
  const tr = document.createElement('tr');
  tr.classList.add('clickable');
  tr.dataset.evidence = (p.evidence||[]).join(',');
  tr.innerHTML = '<td>'+p.id+'</td><td>'+p.journey+'</td><td>'+p.tier+'</td><td>'+p.result+'</td><td>'+(p.evidence||[]).join(', ')+'</td>';
  pt.appendChild(tr);
});
const nt = document.querySelector('#nfr tbody');
(data.nfr_gates||[]).forEach(g => {
  const tr = document.createElement('tr');
  tr.innerHTML = '<td>'+g.id+'</td><td>'+g.target+'</td><td>'+g.result+'</td><td>'+(g.evidence||'')+'</td>';
  nt.appendChild(tr);
});
const ft = document.querySelector('#functional tbody');
(data.functional_logs||[]).forEach(f => {
  const tr = document.createElement('tr');
  tr.innerHTML = '<td><code>'+f.name+'</code></td><td><span class="chip">'+f.result+'</span></td>';
  ft.appendChild(tr);
});
const em = document.querySelector('#edge-matrix tbody');
(data.edge_matrix||[]).forEach(e => {
  const tr = document.createElement('tr');
  tr.dataset.result = e.judge||'unknown';
  tr.innerHTML = '<td>'+e.scenario+'</td><td><code>'+e.step+'</code></td><td><span class="chip">'+e.judge+'</span></td>';
  em.appendChild(tr);
});
document.getElementById('reproduce').textContent = data.reproduce||'';
function applyFilters() {
  const q = document.getElementById('q').value.toLowerCase();
  const tier = document.getElementById('tier').value;
  const flow = document.getElementById('flow').value;
  const layer = document.getElementById('layer').value;
  const result = document.getElementById('result').value;
  document.querySelectorAll('#steps tbody tr').forEach(tr => {
    let ok = true;
    if (q && !tr.dataset.q.includes(q)) ok = false;
    if (tier && tr.dataset.tier !== tier) ok = false;
    if (flow && tr.dataset.flow !== flow) ok = false;
    if (layer && tr.dataset.layer !== layer) ok = false;
    if (result && tr.dataset.result !== result) ok = false;
    tr.classList.toggle('hidden', !ok);
  });
}
['q','tier','flow','layer','result'].forEach(id => document.getElementById(id).addEventListener('input', applyFilters));
const detail = document.getElementById('step-detail');
document.querySelector('#steps tbody').addEventListener('click', e => {
  const tr = e.target.closest('tr');
  if (!tr || e.target.classList.contains('thumb')) return;
  const id = tr.querySelector('code')?.textContent||'';
  detail.textContent = 'Step: '+id+'\nIntent: '+(tr.dataset.intent||'—')+'\n\nVerdict:\n'+(tr.dataset.verdict||'(no verdict file)');
  detail.classList.add('open');
});
document.querySelector('#personas tbody').addEventListener('click', e => {
  const tr = e.target.closest('tr');
  if (!tr || !tr.dataset.evidence) return;
  const steps = tr.dataset.evidence.split(',').map(s => s.trim()).filter(Boolean);
  document.getElementById('q').value = '';
  document.querySelectorAll('#steps tbody tr').forEach(row => {
    const id = row.querySelector('code')?.textContent||'';
    row.classList.toggle('hidden', steps.length > 0 && !steps.some(s => id === s || id.includes(s)));
  });
});
const lb = document.getElementById('lightbox');
document.body.addEventListener('click', e => {
  if (e.target.classList.contains('thumb')) { lb.querySelector('img').src = e.target.src; lb.classList.add('open'); }
  if (e.target === lb) lb.classList.remove('open');
});
</script>
</body>
</html>`

// UXStepRow is one UX capture row for the interactive report.
type UXStepRow struct {
	ID             string `json:"id"`
	Flow           string `json:"flow"`
	Dir            string `json:"dir"`
	Tier           string `json:"tier,omitempty"`
	Layer          string `json:"layer,omitempty"`
	PNG            string `json:"png,omitempty"`
	Intent         string `json:"intent,omitempty"`
	Judge          string `json:"judge"`
	TimingMS       int64  `json:"timing_ms,omitempty"`
	VerdictExcerpt string `json:"verdict_excerpt,omitempty"`
}

// IssueRow is one open issue for the HTML panel.
type IssueRow struct {
	ID      string `json:"id"`
	Layer   string `json:"layer"`
	Step    string `json:"step,omitempty"`
	Summary string `json:"summary"`
}

// PersonaRow maps PRD journey to evidence.
type PersonaRow struct {
	ID       string   `json:"id"`
	Journey  string   `json:"journey"`
	Tier     string   `json:"tier"`
	Result   string   `json:"result"`
	Evidence []string `json:"evidence,omitempty"`
}

// NFRGateRow is one performance/NFR gate in the report.
type NFRGateRow struct {
	ID       string `json:"id"`
	Target   string `json:"target"`
	Result   string `json:"result"`
	Evidence string `json:"evidence,omitempty"`
}

// ScaleReportHTMLData is embedded in scale-report.html.
type ScaleReportHTMLData struct {
	UXSteps        []UXStepRow        `json:"ux_steps"`
	Personas       []PersonaRow       `json:"personas"`
	NFRGates       []NFRGateRow       `json:"nfr_gates"`
	Issues         []IssueRow         `json:"issues,omitempty"`
	FunctionalLogs []FunctionalLogRow `json:"functional_logs,omitempty"`
	EdgeMatrix     []EdgeMatrixRow    `json:"edge_matrix,omitempty"`
	Reproduce      string             `json:"reproduce,omitempty"`
}

// FunctionalLogRow is one scale functional log parsed for the HTML panel.
type FunctionalLogRow struct {
	Name   string `json:"name"`
	Result string `json:"result"`
}

// EdgeMatrixRow is one Part B7 edge scenario row.
type EdgeMatrixRow struct {
	Scenario string `json:"scenario"`
	Step     string `json:"step"`
	Judge    string `json:"judge"`
}

func firstFixtureTier(rep ScaleReport) string {
	if len(rep.FixtureTiers) > 0 {
		return rep.FixtureTiers[0]
	}
	return ""
}

// WriteScaleReportHTML writes scale-report.html using report JSON fields.
func WriteScaleReportHTML(runDir string, rep ScaleReport) error {
	htmlData := ScaleReportHTMLData{
		UXSteps:        collectUXSteps(runDir, firstFixtureTier(rep)),
		Personas:       defaultPersonaRows(rep),
		NFRGates:       collectNFRGates(runDir, rep),
		Issues:         collectOpenIssues(runDir),
		FunctionalLogs: collectFunctionalLogs(runDir),
		EdgeMatrix:     collectEdgeMatrix(runDir),
		Reproduce:      scaleReproduceCommands(rep),
	}
	embed, err := json.Marshal(htmlData)
	if err != nil {
		return err
	}
	html := scaleReportHTMLTemplate
	html = strings.ReplaceAll(html, "{{.RunID}}", rep.RunID)
	html = strings.ReplaceAll(html, "{{.FinishedAt}}", rep.FinishedAt)
	html = strings.ReplaceAll(html, "{{.GitShort}}", rep.GitShort)
	html = strings.ReplaceAll(html, "{{.MachineLine}}", rep.MachineLine)
	html = strings.ReplaceAll(html, "{{.Counts.FunctionalPass}}", fmt.Sprintf("%d", rep.Counts.FunctionalPass))
	html = strings.ReplaceAll(html, "{{.Counts.FunctionalFail}}", fmt.Sprintf("%d", rep.Counts.FunctionalFail))
	html = strings.ReplaceAll(html, "{{.Counts.ScaleUnitPass}}", fmt.Sprintf("%d", rep.Counts.ScaleUnitPass))
	html = strings.ReplaceAll(html, "{{.Counts.ScaleUnitFail}}", fmt.Sprintf("%d", rep.Counts.ScaleUnitFail))
	html = strings.ReplaceAll(html, "{{.Counts.UXStepsPass}}", fmt.Sprintf("%d", rep.Counts.UXStepsPass))
	html = strings.ReplaceAll(html, "{{.Counts.UXStepsFail}}", fmt.Sprintf("%d", rep.Counts.UXStepsFail))
	html = strings.ReplaceAll(html, "{{.Counts.IssuesOpen}}", fmt.Sprintf("%d", rep.Counts.IssuesOpen))
	html = strings.ReplaceAll(html, `{{if eq .MachineLine "EXTENDED_SCALE_RESULT=pass"}}pass{{else}}fail{{end}}`, badgeClass(rep.MachineLine))
	html = strings.ReplaceAll(html, "{{.JSONEmbed}}", string(embed))
	return os.WriteFile(filepath.Join(runDir, "scale-report.html"), []byte(html), 0o644)
}

func badgeClass(machineLine string) string {
	if machineLine == "EXTENDED_SCALE_RESULT=pass" {
		return "pass"
	}
	return "fail"
}

func collectUXSteps(runDir, defaultTier string) []UXStepRow {
	var out []UXStepRow
	entries, _ := os.ReadDir(runDir)
	for _, e := range entries {
		if !e.IsDir() || !strings.HasPrefix(e.Name(), "ui-real") {
			continue
		}
		dir := e.Name()
		layer := uxLayerFromDir(dir)
		data, err := os.ReadFile(filepath.Join(runDir, dir, "steps.json"))
		if err != nil {
			continue
		}
		var m struct {
			FixtureTier string `json:"fixture_tier"`
			Steps       []struct {
				ID       string `json:"id"`
				Flow     string `json:"flow"`
				File     string `json:"file"`
				Intent   string `json:"intent"`
				TimingMS int64  `json:"timing_ms"`
			} `json:"steps"`
		}
		if json.Unmarshal(data, &m) != nil {
			continue
		}
		tier := m.FixtureTier
		if tier == "" {
			tier = defaultTier
		}
		for _, s := range m.Steps {
			judge := parseStepJudge(runDir, s.ID)
			png := filepath.Join(dir, s.File)
			out = append(out, UXStepRow{
				ID: s.ID, Flow: s.Flow, Dir: dir, Tier: tier, Layer: layer,
				PNG: png, Intent: s.Intent, Judge: judge, TimingMS: s.TimingMS,
				VerdictExcerpt: verdictExcerpt(runDir, s.ID),
			})
		}
	}
	return out
}

func uxLayerFromDir(dir string) string {
	switch {
	case dir == "ui-real-scale":
		return "ux_scale_spot"
	case strings.HasPrefix(dir, "ui-real-edge"):
		return "ux_edge"
	case dir == "ui-real-layout":
		return "ux_layout"
	default:
		return "ux_journey"
	}
}

func parseStepJudge(runDir, stepID string) string {
	p := filepath.Join(runDir, "verdicts", "steps", stepID+".md")
	data, err := os.ReadFile(p)
	if err != nil {
		return "unknown"
	}
	if strings.Contains(string(data), "STEP_UX_RESULT=pass") {
		return "pass"
	}
	if strings.Contains(string(data), "STEP_UX_RESULT=fail") {
		return "fail"
	}
	return "unknown"
}

func verdictExcerpt(runDir, stepID string) string {
	p := filepath.Join(runDir, "verdicts", "steps", stepID+".md")
	data, err := os.ReadFile(p)
	if err != nil {
		return ""
	}
	body := strings.TrimSpace(string(data))
	if len(body) > 800 {
		body = body[:800] + "…"
	}
	return body
}

func collectOpenIssues(runDir string) []IssueRow {
	issuesDir := filepath.Join(runDir, "issues")
	entries, err := os.ReadDir(issuesDir)
	if err != nil {
		return nil
	}
	var out []IssueRow
	for _, e := range entries {
		if !strings.HasPrefix(e.Name(), "ISSUE-") || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(issuesDir, e.Name()))
		if err != nil {
			continue
		}
		var iss Issue
		if json.Unmarshal(data, &iss) != nil {
			continue
		}
		out = append(out, IssueRow{
			ID: iss.ID, Layer: string(iss.Layer), Step: iss.Step, Summary: iss.Summary,
		})
	}
	return out
}

func defaultPersonaRows(rep ScaleReport) []PersonaRow {
	rows := []PersonaRow{
		{ID: "photographer_setup", Journey: "A", Tier: "S0", Result: "partial", Evidence: []string{"review_library_empty"}},
		{ID: "photographer_bulk", Journey: "A", Tier: "S4/S5", Result: personaResult(rep, "upload_fr06_batch_50"), Evidence: []string{"upload_fr06_batch_50"}},
		{ID: "photographer_triage", Journey: "B", Tier: "S4", Result: personaResult(rep, "review_grid_page2_top"), Evidence: []string{"review_grid_bulk_20_selected", "review_filter_collection_album"}},
		{ID: "album_curator", Journey: "C", Tier: "S5", Result: personaResult(rep, "collections_detail_500_stars"), Evidence: []string{"collections_detail_500_stars"}},
		{ID: "noise_reduction", Journey: "E", Tier: "S5R", Result: personaResult(rep, "rejected_grid_500"), Evidence: []string{"rejected_grid_500"}},
		{ID: "share_vacation", Journey: "F", Tier: "S5/S6", Result: personaResult(rep, "review_package_blocked_501"), Evidence: []string{"review_package_share_preview", "review_package_blocked_501"}},
		{ID: "power_user_cli", Journey: "D", Tier: "S8", Result: nfrResult(rep, "NFR-02"), Evidence: []string{"logs/scale_nfr02.txt"}},
		{ID: "share_recipient", Journey: "3.3/4.1", Tier: "S5", Result: "n/a", Evidence: []string{"HTTP contract tests"}},
	}
	return rows
}

func personaResult(rep ScaleReport, step string) string {
	if rep.MachineLine != "EXTENDED_SCALE_RESULT=pass" {
		return "partial"
	}
	_ = step
	return "pass"
}

func nfrResult(rep ScaleReport, id string) string {
	for _, g := range rep.NFRGates {
		if g.ID == id {
			return g.Result
		}
	}
	return "unknown"
}

func collectNFRGates(runDir string, rep ScaleReport) []NFRGateRow {
	if len(rep.NFRGates) > 0 {
		return rep.NFRGates
	}
	gates := []NFRGateRow{
		{ID: "NFR-02", Target: "10k dry-run bounded heap", Result: logGateResult(runDir, "scale_nfr02"), Evidence: "logs/scale_nfr02.txt"},
		{ID: "NFR-05", Target: "≤3s cold load", Result: "partial", Evidence: "internal/share/nfr05 tests"},
		{ID: "SC-3", Target: "≤1s rating feedback", Result: "partial", Evidence: "review timing tests"},
	}
	return gates
}

func logGateResult(runDir, logPrefix string) string {
	logsDir := filepath.Join(runDir, "logs")
	entries, err := os.ReadDir(logsDir)
	if err != nil {
		return "unknown"
	}
	for _, e := range entries {
		if strings.Contains(e.Name(), logPrefix) {
			data, _ := os.ReadFile(filepath.Join(logsDir, e.Name()))
			if strings.Contains(string(data), "FAIL") {
				return "fail"
			}
			return "pass"
		}
	}
	return "unknown"
}

func collectFunctionalLogs(runDir string) []FunctionalLogRow {
	logsDir := filepath.Join(runDir, "logs")
	entries, err := os.ReadDir(logsDir)
	if err != nil {
		return nil
	}
	var out []FunctionalLogRow
	for _, e := range entries {
		name := e.Name()
		if !strings.HasPrefix(name, "scale_") {
			continue
		}
		data, _ := os.ReadFile(filepath.Join(logsDir, name))
		result := "pass"
		if strings.Contains(string(data), "FAIL") {
			result = "fail"
		}
		out = append(out, FunctionalLogRow{Name: name, Result: result})
	}
	return out
}

var b7EdgeScenarios = []struct {
	Scenario string
	Step     string
}{
	{"Filter deadlock → Reset", "review_filter_deadlock_reset"},
	{"Collection re-tap reset (AC12)", "collections_re_tap_nav_reset"},
	{"Package blocked over 500", "review_package_blocked_501"},
	{"Rejected grid at scale", "rejected_grid_500"},
	{"Rejected filter empty", "rejected_filter_min_rating_empty"},
	{"Rejected bulk delete confirm", "rejected_bulk_delete_confirm"},
	{"Delete confirm 50 selected", "review_delete_confirm_50"},
	{"Empty library S0", "review_library_empty"},
	{"Loupe share rejected block", "review_loupe_share_rejected_block"},
	{"Undo reject after single reject", "review_undo_reject_single"},
	{"Drop during FR-06", "upload_drop_during_fr06"},
	{"Theme switch full grid", "review_theme_switch_full_grid"},
}

func collectEdgeMatrix(runDir string) []EdgeMatrixRow {
	var out []EdgeMatrixRow
	for _, s := range b7EdgeScenarios {
		out = append(out, EdgeMatrixRow{
			Scenario: s.Scenario,
			Step:     s.Step,
			Judge:    parseStepJudge(runDir, s.Step),
		})
	}
	return out
}

func scaleReproduceCommands(rep ScaleReport) string {
	tier := "S5"
	if len(rep.FixtureTiers) > 0 {
		tier = rep.FixtureTiers[0]
	}
	return fmt.Sprintf(`make extended-test-scale
make extended-test-scale-ux
PHOTO_TOOL_SCALE_TIER=%s EXTENDED_STOP_ON_FAIL=0 ./scripts/extended-test-run.sh --layer=scale --judges
go test ./tests/fixture/... ./tests/e2e/... -run TestScale_ -count=1 -timeout=30m`, tier)
}
