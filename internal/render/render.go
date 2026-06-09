package render

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Rulunchic/ObGoData/internal/analyze"
	"github.com/Rulunchic/ObGoData/internal/scanner"
)

//go:embed theme.css
var themeCSS string

type pageData struct {
	VaultName      string
	Stats          *analyze.Stats
	NotesJSON      string
	WordsJSON      string
	CharsJSON      string
	LemmasJSON     string
	DatesJSON      string
	HoursJSON      string
	WeekdayJSON    string
	SizeByDirJSON  string
	FilesByDirJSON string
	LinkTypesJSON  string
	AttachTypesJSON string
	LargestNotes   []scanner.Note
}

func Generate(s *analyze.Stats, vaultPath string) (string, error) {
	outDir := filepath.Join(os.TempDir(), "obgodata-"+fmt.Sprint(time.Now().Unix()))
	err := os.MkdirAll(outDir, 0755)
	if err != nil {
		return "", err
	}

	vaultName := vaultPath
	if short := filepath.Base(vaultPath); short != "" {
		vaultName = short
	}

	d := &pageData{
		VaultName:    vaultName,
		Stats:        s,
		LargestNotes: s.LargestNotes,
	}

	d.DatesJSON = buildDatesJSON(s.NotesByDate)
	d.HoursJSON = buildHoursJSON(s.NotesByHour)
	d.WeekdayJSON = buildWeekdayJSON(s.NotesByWeekday)
	d.WordsJSON = buildWordFreqJSON(s.WordFreq)
	d.CharsJSON = buildCharFreqJSON(s.CharFreq)
	d.LemmasJSON = buildWordFreqJSON(s.LemmaFreq)
	d.NotesJSON = buildNotesJSON(s.WordsPerNote)
	d.SizeByDirJSON = buildMapJSON(s.SizeByDir)
	d.FilesByDirJSON = buildMapJSON(s.FilesByDir)
	d.LinkTypesJSON = buildMapJSON(s.LinkTypes)
	d.AttachTypesJSON = buildMapJSON(s.AttachTypes)

	html := renderHTML(d)

	err = os.WriteFile(filepath.Join(outDir, "index.html"), []byte(html), 0644)
	if err != nil {
		return "", err
	}

	err = os.WriteFile(filepath.Join(outDir, "theme.css"), []byte(themeCSS), 0644)
	if err != nil {
		return "", err
	}

	return outDir, nil
}

func renderHTML(d *pageData) string {
	s := d.Stats

	var tableRows string
	for _, n := range d.LargestNotes {
		tableRows += fmt.Sprintf("<tr><td>%s</td><td>%s</td><td class=\"num\">%d</td><td class=\"num\">%s</td><td>%s</td></tr>\n",
			escHTML(n.Name), escHTML(n.Path), n.WordCount, formatSize(n.Size),
			n.ModTime.Format("2006-01-02"))
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="ru">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width,initial-scale=1.0">
<title>ObGoData — %s</title>
<script src="https://cdn.jsdelivr.net/npm/chart.js@4"></script>
<script src="https://cdn.jsdelivr.net/npm/chartjs-plugin-zoom@2"></script>
<style>%s</style>
</head>
<body>

<nav>ObGoData <span class="muted">— vault: %s</span></nav>

<main id="dashboard">

<section class="stats-grid">
  <div class="stat-card"><div class="value">%d</div><div class="label">Notes</div></div>
  <div class="stat-card"><div class="value">%s</div><div class="label">Total words</div></div>
  <div class="stat-card"><div class="value">%d</div><div class="label">Links</div></div>
  <div class="stat-card"><div class="value">%d</div><div class="label">Attachments</div></div>
  <div class="stat-card"><div class="value">%s</div><div class="label">Total size</div></div>
  <div class="stat-card"><div class="value">%d</div><div class="label">Directories</div></div>
</section>

<section class="chart-grid">

  <div class="chart-panel">
    <div class="chart-hdr"><h2>Notes over time</h2><div class="chart-actions"><button class="cbtn" onclick="resetZoom('chartDates')">↺</button><button class="cbtn" onclick="toggleFull('chartDates')">+</button></div></div>
    <canvas id="chartDates"></canvas>
  </div>

  <div class="chart-panel">
    <div class="chart-hdr"><h2>Activity by hour</h2><div class="chart-actions"><button class="cbtn" onclick="resetZoom('chartHours')">↺</button><button class="cbtn" onclick="toggleFull('chartHours')">+</button></div></div>
    <canvas id="chartHours"></canvas>
  </div>

  <div class="chart-panel">
    <div class="chart-hdr"><h2>Words per note</h2><div class="chart-actions"><button class="cbtn" onclick="resetZoom('chartNotes')">↺</button><button class="cbtn" onclick="toggleFull('chartNotes')">+</button></div></div>
    <canvas id="chartNotes"></canvas>
  </div>

  <div class="chart-panel">
    <div class="chart-hdr"><h2>Activity by weekday</h2><div class="chart-actions"><button class="cbtn" onclick="resetZoom('chartWeekday')">↺</button><button class="cbtn" onclick="toggleFull('chartWeekday')">+</button></div></div>
    <canvas id="chartWeekday"></canvas>
  </div>

  <div class="chart-panel">
    <div class="chart-hdr"><h2>Top words</h2><div class="chart-actions"><button class="cbtn" onclick="toggleSort('chartWords')">↓↑</button><button class="cbtn" onclick="resetZoom('chartWords')">↺</button><button class="cbtn" onclick="toggleFull('chartWords')">+</button></div></div>
    <canvas id="chartWords"></canvas>
  </div>

  <div class="chart-panel">
    <div class="chart-hdr"><h2>Character frequency</h2><div class="chart-actions"><button class="cbtn" onclick="toggleSort('chartChars')">↓↑</button><button class="cbtn" onclick="resetZoom('chartChars')">↺</button><button class="cbtn" onclick="toggleFull('chartChars')">+</button></div></div>
    <canvas id="chartChars"></canvas>
  </div>

  <div class="chart-panel">
    <div class="chart-hdr"><h2>Lemma frequency</h2><div class="chart-actions"><button class="cbtn" onclick="toggleSort('chartLemmas')">↓↑</button><button class="cbtn" onclick="resetZoom('chartLemmas')">↺</button><button class="cbtn" onclick="toggleFull('chartLemmas')">+</button></div></div>
    <canvas id="chartLemmas"></canvas>
  </div>

  <div class="chart-panel">
    <div class="chart-hdr"><h2>File size by directory</h2><div class="chart-actions"><button class="cbtn" onclick="toggleSort('chartSizeByDir')">↓↑</button><button class="cbtn" onclick="resetZoom('chartSizeByDir')">↺</button><button class="cbtn" onclick="toggleFull('chartSizeByDir')">+</button></div></div>
    <canvas id="chartSizeByDir"></canvas>
  </div>

  <div class="chart-panel">
    <div class="chart-hdr"><h2>Files by directory</h2><div class="chart-actions"><button class="cbtn" onclick="toggleSort('chartFilesByDir')">↓↑</button><button class="cbtn" onclick="resetZoom('chartFilesByDir')">↺</button><button class="cbtn" onclick="toggleFull('chartFilesByDir')">+</button></div></div>
    <canvas id="chartFilesByDir"></canvas>
  </div>

  <div class="chart-panel">
    <div class="chart-hdr"><h2>Link types</h2><div class="chart-actions"><button class="cbtn" onclick="resetZoom('chartLinks')">↺</button><button class="cbtn" onclick="toggleFull('chartLinks')">+</button></div></div>
    <canvas id="chartLinks"></canvas>
  </div>

  <div class="chart-panel">
    <div class="chart-hdr"><h2>Attachment types</h2><div class="chart-actions"><button class="cbtn" onclick="resetZoom('chartAttachs')">↺</button><button class="cbtn" onclick="toggleFull('chartAttachs')">+</button></div></div>
    <canvas id="chartAttachs"></canvas>
  </div>

</section>

<section>
  <h2>Largest notes</h2>
  <div class="table-wrap">
  <table>
    <thead><tr><th>Name</th><th>Path</th><th>Words</th><th>Size</th><th>Modified</th></tr></thead>
    <tbody>%s</tbody></table></div></section>

</main>

<div class="footer">Generated by ObGoData</div>

<script>
const NOTES = %s;
const WORDS = %s;
const CHARS = %s;
const LEMMAS = %s;
const DATES = %s;
const HOURS = %s;
const WEEKDAYS = %s;
const SIZEBYDIR = %s;
const FILESBYDIR = %s;
const LINKTYPES = %s;
const ATTACHTYPES = %s;

%s
</script>
</body>
</html>`,
		d.VaultName, themeCSS, d.VaultName,
		s.TotalNotes, formatInt(s.TotalWords), s.TotalLinks, s.TotalAttachs,
		formatSize(s.TotalSize), s.DirCount,
		tableRows,
		d.NotesJSON, d.WordsJSON, d.CharsJSON, d.LemmasJSON, d.DatesJSON,
		d.HoursJSON, d.WeekdayJSON, d.SizeByDirJSON, d.FilesByDirJSON,
		d.LinkTypesJSON, d.AttachTypesJSON,
		chartJS,
	)
}

const chartJS = `if (typeof ChartZoom !== 'undefined') { try { Chart.register(ChartZoom.Zoom || ChartZoom); } catch(e) {} }

var charts = {};

function hsv(i,n) {
  var h = (i*0.618033988749895+0.3)%1;
  return 'hsl('+Math.floor(h*360)+','+(40+20*(i%2))+'%,'+(55+10*(i%2))+'%)';
}

var zOpts = {
  zoom:{ drag:{enabled:true,backgroundColor:'rgba(124,184,232,0.2)',borderColor:'#7cb8e8',borderWidth:1 },
         wheel:{enabled:true},pinch:{enabled:true},mode:'xy' },
  pan:{ enabled:true,mode:'xy' }
};

function barOpts() {
  return { responsive:true, maintainAspectRatio:false,
    plugins:{legend:{display:false},zoom:zOpts},
    scales:{ x:{ ticks:{color:'#999',font:{size:10}},grid:{display:false} },
             y:{ ticks:{color:'#999',font:{size:10}},beginAtZero:true,grid:{color:'rgba(255,255,255,0.05)'} } } };
}

function horizBarOpts() {
  return { indexAxis:'y', responsive:true, maintainAspectRatio:false,
    plugins:{legend:{display:false},zoom:zOpts},
    scales:{ x:{ ticks:{color:'#999',font:{size:10}},grid:{color:'rgba(255,255,255,0.05)'} },
             y:{ ticks:{color:'#999',font:{size:10}},grid:{display:false} } } };
}

function doughnutOpts() {
  return { responsive:true, maintainAspectRatio:false,
    plugins:{ legend:{ labels:{color:'#999',font:{size:11}} } } };
}

function addChart(id, config) {
  var el = document.getElementById(id);
  if (!el) return;
  charts[id] = { chart: new Chart(el, config), config: config };
}

function defaultColors(n) {
  var c = [];
  for (var i=0;i<n;i++) c.push(hsv(i,n));
  return c;
}

function initExpandable(id, labels, allData, topN) {
  var entry = charts[id];
  if (!entry) return;
  entry._fullLabels = labels;
  entry._fullData = allData;
  entry._topN = topN;
  var topLabels = labels.slice(-topN);
  var topData = allData.slice(-topN);
  entry.chart.data.labels = topLabels;
  entry.chart.data.datasets[0].data = topData;
  entry.chart.update();
}

function toggleSort(id) {
  var entry = charts[id];
  if (!entry) return;
  var chart = entry.chart;
  var ds = chart.data.datasets[0];
  chart.data.labels.reverse();
  ds.data.reverse();
  if (entry._fullLabels) { entry._fullLabels.reverse(); entry._fullData.reverse(); }
  chart.update();
}

function toggleFull(id) {
  var panel = document.getElementById(id).closest('.chart-panel');
  if (!panel) return;
  var entering = !panel.classList.contains('fullscreen');
  panel.classList.toggle('fullscreen');
  var btn = panel.querySelector('.chart-actions .cbtn:last-child');
  if (btn) btn.textContent = entering ? '−' : '+';
  var entry = charts[id];
  if (entry && entry._fullLabels) {
    if (entering) {
      entry.chart.data.labels = entry._fullLabels;
      entry.chart.data.datasets[0].data = entry._fullData;
    } else {
      entry.chart.data.labels = entry._fullLabels.slice(-entry._topN);
      entry.chart.data.datasets[0].data = entry._fullData.slice(-entry._topN);
    }
    entry.chart.update();
  }
  setTimeout(function(){
    if (entry) entry.chart.resize();
  }, 100);
}

function resetZoom(id) {
  var entry = charts[id];
  if (entry) entry.chart.resetZoom();
}

(function(){
  var dates = DATES;
  var labels = Object.keys(dates).sort();
  if (labels.length===0) { document.getElementById('chartDates').parentElement.style.display='none'; return; }
  var months = ['Jan','Feb','Mar','Apr','May','Jun','Jul','Aug','Sep','Oct','Nov','Dec'];
  addChart('chartDates', {
    type:'bar', data:{ labels:labels.map(function(d){var dt=new Date(d);return dt.getDate()+' '+months[dt.getMonth()]+' '+dt.getFullYear();}), datasets:[{label:'Notes',data:labels.map(function(d){return dates[d];}),backgroundColor:hsv(0,10),borderRadius:2}] },
    options: barOpts()
  });
})();

(function(){
  addChart('chartHours', {
    type:'bar', data:{ labels:Array.from({length:24},function(_,i){return i+':00';}), datasets:[{label:'Notes',data:HOURS,backgroundColor:hsv(1,10),borderRadius:2}] },
    options: barOpts()
  });
})();

(function(){
  var data = NOTES;
  if (data.length===0) { document.getElementById('chartNotes').parentElement.style.display='none'; return; }
  var max = Math.max.apply(null, data);
  var step = Math.max(1, Math.ceil(max/20));
  var buckets = {};
  for (var i=0;i<=max;i+=step) { var l=i+'-'+(i+step); buckets[l]=0; }
  data.forEach(function(v){ var idx=Math.min(Math.floor(v/step)*step,Math.floor(max/step)*step); var l=idx+'-'+(idx+step); buckets[l]=(buckets[l]||0)+1; });
  addChart('chartNotes', {
    type:'bar', data:{ labels:Object.keys(buckets), datasets:[{label:'Notes',data:Object.values(buckets),backgroundColor:hsv(2,10),borderRadius:2}] },
    options: barOpts()
  });
})();

(function(){
  addChart('chartWeekday', {
    type:'bar', data:{ labels:['Sun','Mon','Tue','Wed','Thu','Fri','Sat'], datasets:[{label:'Notes',data:WEEKDAYS,backgroundColor:hsv(3,10),borderRadius:2}] },
    options: barOpts()
  });
})();

(function(){
  var raw = WORDS;
  if (raw.length===0) { document.getElementById('chartWords').parentElement.style.display='none'; return; }
  var labels = raw.map(function(w){return w.w;}).reverse();
  var data = raw.map(function(w){return w.c;}).reverse();
  addChart('chartWords', {
    type:'bar', data:{ labels:labels, datasets:[{label:'Count',data:data,backgroundColor:hsv(4,10),borderRadius:2}] },
    options: horizBarOpts()
  });
  initExpandable('chartWords', labels, data, 30);
})();

(function(){
  var raw = CHARS;
  if (raw.length===0) { document.getElementById('chartChars').parentElement.style.display='none'; return; }
  var labels = raw.map(function(c){return c.ch;}).reverse();
  var data = raw.map(function(c){return c.c;}).reverse();
  addChart('chartChars', {
    type:'bar', data:{ labels:labels, datasets:[{label:'Count',data:data,backgroundColor:hsv(5,10),borderRadius:2}] },
    options: horizBarOpts()
  });
  initExpandable('chartChars', labels, data, 30);
})();

(function(){
  var raw = LEMMAS;
  if (raw.length===0) { document.getElementById('chartLemmas').parentElement.style.display='none'; return; }
  var labels = raw.map(function(l){return l.w;}).reverse();
  var data = raw.map(function(l){return l.c;}).reverse();
  addChart('chartLemmas', {
    type:'bar', data:{ labels:labels, datasets:[{label:'Count',data:data,backgroundColor:hsv(8,10),borderRadius:2}] },
    options: horizBarOpts()
  });
  initExpandable('chartLemmas', labels, data, 30);
})();

(function(){
  var raw = SIZEBYDIR;
  var keys = Object.keys(raw).sort(function(a,b){return raw[b]-raw[a];});
  if (keys.length===0) { document.getElementById('chartSizeByDir').parentElement.style.display='none'; return; }
  function fmtBytes(v) {
    if(v>=1073741824) return (v/1073741824).toFixed(1)+' GB';
    if(v>=1048576) return (v/1048576).toFixed(1)+' MB';
    if(v>=1024) return (v/1024).toFixed(1)+' KB';
    return v+' B';
  }
  var labels = keys.reverse();
  var data = keys.map(function(k){return raw[k];}).reverse();
  addChart('chartSizeByDir', {
    type:'bar', data:{ labels:labels, datasets:[{label:'Size',data:data,backgroundColor:hsv(6,10),borderRadius:2}] },
    options: { indexAxis:'y', responsive:true, maintainAspectRatio:false,
      plugins:{legend:{display:false}, zoom:zOpts, tooltip:{callbacks:{label:function(ctx){return fmtBytes(ctx.raw);}}}},
      scales:{ x:{ ticks:{color:'#999',font:{size:10},callback:function(v){return fmtBytes(v);}},grid:{color:'rgba(255,255,255,0.05)'} },
               y:{ ticks:{color:'#999',font:{size:10}},grid:{display:false} } } }
  });
  initExpandable('chartSizeByDir', labels, data, 15);
})();

(function(){
  var raw = FILESBYDIR;
  var keys = Object.keys(raw).sort(function(a,b){return raw[b]-raw[a];});
  if (keys.length===0) { document.getElementById('chartFilesByDir').parentElement.style.display='none'; return; }
  var labels = keys.reverse();
  var data = keys.map(function(k){return raw[k];}).reverse();
  addChart('chartFilesByDir', {
    type:'bar', data:{ labels:labels, datasets:[{label:'Files',data:data,backgroundColor:hsv(7,10),borderRadius:2}] },
    options: horizBarOpts()
  });
  initExpandable('chartFilesByDir', labels, data, 15);
})();

(function(){
  var data = LINKTYPES;
  var keys = Object.keys(data).filter(function(k){return k!=='total';});
  if (keys.length===0) { document.getElementById('chartLinks').parentElement.style.display='none'; return; }
  addChart('chartLinks', {
    type:'doughnut', data:{ labels:keys, datasets:[{data:keys.map(function(k){return data[k];}),backgroundColor:defaultColors(keys.length),borderWidth:0}] },
    options: doughnutOpts()
  });
})();

(function(){
  var data = ATTACHTYPES;
  var keys = Object.keys(data);
  if (keys.length===0) { document.getElementById('chartAttachs').parentElement.style.display='none'; return; }
  addChart('chartAttachs', {
    type:'doughnut', data:{ labels:keys, datasets:[{data:keys.map(function(k){return data[k];}),backgroundColor:defaultColors(keys.length),borderWidth:0}] },
    options: doughnutOpts()
  });
})();`

func buildDatesJSON(m map[string]int) string {
	return buildMapJSON(m)
}

func buildHoursJSON(h [24]int) string {
	var parts []string
	for _, v := range h {
		parts = append(parts, fmt.Sprint(v))
	}
	return "[" + strings.Join(parts, ",") + "]"
}

func buildWeekdayJSON(w [7]int) string {
	var parts []string
	for _, v := range w {
		parts = append(parts, fmt.Sprint(v))
	}
	return "[" + strings.Join(parts, ",") + "]"
}

func buildNotesJSON(words []int) string {
	var parts []string
	for _, v := range words {
		parts = append(parts, fmt.Sprint(v))
	}
	return "[" + strings.Join(parts, ",") + "]"
}

func buildWordFreqJSON(freq []analyze.WordCount) string {
	var items []string
	for _, f := range freq {
		items = append(items, fmt.Sprintf(`{"w":%q,"c":%d}`, escJSON(f.Word), f.Count))
	}
	return "[" + strings.Join(items, ",") + "]"
}

func buildCharFreqJSON(freq []analyze.CharCount) string {
	var items []string
	for _, f := range freq {
		items = append(items, fmt.Sprintf(`{"ch":%q,"c":%d}`, escJSON(f.Char), f.Count))
	}
	return "[" + strings.Join(items, ",") + "]"
}

func buildMapJSON(m interface{}) string {
	switch v := m.(type) {
	case map[string]int:
		if len(v) == 0 {
			return "{}"
		}
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		var pairs []string
		for _, k := range keys {
			pairs = append(pairs, fmt.Sprintf(`%q:%d`, escJSON(k), v[k]))
		}
		return "{" + strings.Join(pairs, ",") + "}"
	case map[string]int64:
		if len(v) == 0 {
			return "{}"
		}
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		var pairs []string
		for _, k := range keys {
			pairs = append(pairs, fmt.Sprintf(`%q:%d`, escJSON(k), v[k]))
		}
		return "{" + strings.Join(pairs, ",") + "}"
	}
	return "{}"
}

func escJSON(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "\\r")
	s = strings.ReplaceAll(s, "\t", "\\t")
	return s
}

func escHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return s
}

func formatInt(n int) string {
	if n < 1000 {
		return fmt.Sprint(n)
	}
	return fmt.Sprintf("%.1fK", float64(n)/1000)
}

func formatSize(n int64) string {
	if n >= 1073741824 {
		return fmt.Sprintf("%.1f GB", float64(n)/1073741824)
	}
	if n >= 1048576 {
		return fmt.Sprintf("%.1f MB", float64(n)/1048576)
	}
	if n >= 1024 {
		return fmt.Sprintf("%.1f KB", float64(n)/1024)
	}
	return fmt.Sprintf("%d B", n)
}
