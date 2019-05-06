// Copyright 2019 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package targetstats provides a statistics for Prometheus targets.
package targetstats

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"sort"
	"strings"

	"github.com/prometheus/prometheus/pkg/textparse"
)

// A TargetAnalyzer is a Prometheus target statistics interface. It computes stats
// about a prometheus target and reports them to the caller.
type TargetAnalyzer struct {
	r io.Reader
}

type TargetStat struct {
	Name  string
	Value float64
}

type TargetStats struct {
	Generic []TargetStat
	Types   []TargetStat
	Series  []TargetStat
}

// New creates a new TargetAnalyzer that reads an input stream of Prometheus target.
func New(r io.Reader) *TargetAnalyzer {
	return &TargetAnalyzer{
		r: r,
	}
}

// Analyze performs a statistics analysis on the target, returning statistics
// about metrics/labels found in the exporter data
func (l *TargetAnalyzer) Analyze(sortBy string) (TargetStats, error) {
	stats := TargetStats{}

	typeStats := make(map[string]int)
	genericStats := make(map[string]int)
	serieStats := make(map[string]int)

	b, err := ioutil.ReadAll(l.r)
	if err != nil {
		return stats, err
	}

	t := textparse.NewPromParser(b)

	for {
		e, err := t.Next()

		if err != nil {
			break
		}
		switch e {
		case textparse.EntryInvalid:
			return stats, errors.New("Invalid entry during target parsing")

		case textparse.EntryType:
			_, metricType := t.Type()
			// fmt.Println("Type", string(metricName), metricType)
			typeStats[string(metricType)] += 1
			genericStats["Type"] += 1

		case textparse.EntryHelp:
			// metricName, metricHelp := t.Help()
			// fmt.Println("Help", string(metricName), string(metricHelp))
			genericStats["Help"] += 1

		case textparse.EntrySeries:
			metricSeries, _, _ := t.Series()
			s := strings.ToLower(string(metricSeries))

			cutHere := strings.Index(s, "{")
			if cutHere != -1 {
				s = string(s[:cutHere])
			}

			serieStats[s] += 1
			genericStats["Series"] += 1

		case textparse.EntryComment:
			// fmt.Println("Comment", string(t.Comment()))
			// statsTable["Series"] += 1
			genericStats["Comment"] += 1

		case textparse.EntryUnit:
			// metricName, metricUnit := t.Unit()
			// fmt.Println("Unit", string(metricName), string(metricUnit))
			genericStats["Unit"] += 1

		default:
			return stats, fmt.Errorf("Unknown entry type %d", e)
		}
	}

	for n, s := range genericStats {
		stats.Generic = append(stats.Generic, TargetStat{
			Name:  fmt.Sprintf("%s_count", n),
			Value: float64(s),
		})
	}

	for n, s := range typeStats {
		stats.Types = append(stats.Types, TargetStat{
			Name:  fmt.Sprintf("%s_count", n),
			Value: float64(s),
		})
	}

	for n, s := range serieStats {
		stats.Series = append(stats.Series, TargetStat{
			Name:  fmt.Sprintf("%s_count", n),
			Value: float64(s),
		})
	}

	switch sortBy {
	case "value":
		sort.Sort(ByValueDesc(stats.Generic))
		sort.Sort(ByValueDesc(stats.Types))
		sort.Sort(ByValueDesc(stats.Series))

	case "name":
		sort.Sort(ByName(stats.Generic))
		sort.Sort(ByName(stats.Types))
		sort.Sort(ByName(stats.Series))
	}

	return stats, nil
}

// implement sort interface for TargetStat
type ByValueDesc []TargetStat

func (a ByValueDesc) Len() int           { return len(a) }
func (a ByValueDesc) Less(i, j int) bool { return a[i].Value > a[j].Value }
func (a ByValueDesc) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }

// implement sort interface for TargetStat
type ByName []TargetStat

func (a ByName) Len() int           { return len(a) }
func (a ByName) Less(i, j int) bool { return a[i].Name < a[j].Name }
func (a ByName) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
