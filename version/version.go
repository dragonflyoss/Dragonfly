/*
 * Copyright The Dragonfly Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Package version represents the version the project Dragonfly.
package version

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"text/template"

	"github.com/dragonflyoss/Dragonfly/apis/types"
	"github.com/dragonflyoss/Dragonfly/pkg/metricsutils"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	// version is the version of project Dragonfly
	// populate via ldflags
	version string

	// revision is the current git commit revision
	// populate via ldflags
	revision string

	// buildDate is the build date of project Dragonfly
	// populate via ldflags
	buildDate string

	// goVersion is the running program's golang version.
	goVersion = runtime.Version()

	// os is the running program's operating system.
	os = runtime.GOOS

	// arch is the running program's architecture target.
	arch = runtime.GOARCH

	// DFDaemonVersion is the version of dfdaemon.
	DFDaemonVersion = version

	// DFGetVersion is the version of dfget.
	DFGetVersion = version

	// SupernodeVersion is the version of supernode.
	SupernodeVersion = version

	// DFVersion is the global instance of DragonflyVersion.
	DFVersion *types.DragonflyVersion
)

func init() {
	DFVersion = &types.DragonflyVersion{
		BuildDate: buildDate,
		Arch:      arch,
		OS:        os,
		GoVersion: goVersion,
		Version:   version,
	}
}

// versionInfoTmpl contains the template used by Info.
var versionInfoTmpl = `
{{.program}} version  {{.version}}
  Git commit:     {{.revision}}
  Build date:     {{.buildDate}}
  Go version:     {{.goVersion}}
  OS/Arch:        {{.OS}}/{{.Arch}}
`

// Print returns version information.
func Print(program string) string {
	m := map[string]string{
		"program":   program,
		"version":   version,
		"revision":  revision,
		"buildDate": buildDate,
		"goVersion": goVersion,
		"OS":        os,
		"Arch":      arch,
	}
	t := template.Must(template.New("version").Parse(versionInfoTmpl))

	var buf bytes.Buffer
	if err := t.ExecuteTemplate(&buf, "version", m); err != nil {
		panic(err)
	}
	return strings.TrimSpace(buf.String())
}

// NewBuildInfo registers a collector which exports metrics about version and build information.
func NewBuildInfo(program string, registerer prometheus.Registerer) {
	buildInfo := metricsutils.NewGauge(program, "build_info",
		fmt.Sprintf("A metric with a constant '1' value labeled by version, revision, os, "+
			"arch and goversion from which %s was built.", program),
		[]string{"version", "revision", "os", "arch", "goversion"},
		registerer,
	)
	buildInfo.WithLabelValues(version, revision, os, arch, goVersion).Set(1)
}

// Handler returns build information.
func Handler(w http.ResponseWriter, r *http.Request) {
	data, err := json.Marshal(DFVersion)
	if err != nil {
		http.Error(w, fmt.Sprintf("error encoding JSON: %s", err), http.StatusInternalServerError)
	} else if _, err := w.Write(data); err != nil {
		http.Error(w, fmt.Sprintf("error writing the data to the connection: %s", err), http.StatusInternalServerError)
	} else {
		w.WriteHeader(http.StatusOK)
	}
}

// HandlerWithCtx returns build information.
func HandlerWithCtx(context context.Context, w http.ResponseWriter, r *http.Request) (err error) {
	Handler(w, r)
	return
}
