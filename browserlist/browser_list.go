package browserlist

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gernest/gs/ciu/agents"
	"github.com/gernest/gs/e2c"
)

var browserAlias = map[string]string{
	"fx":             "firefox",
	"ff":             "firefox",
	"ios":            "ios_saf",
	"explorer":       "ie",
	"blackberry":     "bb",
	"explorermobile": "ie_mob",
	"operamini":      "op_mini",
	"operamobile":    "op_mob",
	"chromeandroid":  "and_chr",
	"firefoxandroid": "and_ff",
	"ucandroid":      "and_uc",
	"qqandroid":      "and_qq",
}

func browserName(name string) string {
	name = strings.ToLower(name)
	if n, ok := browserAlias[name]; ok {
		return n
	}
	return name
}

type version string

func (v version) eq(v2 version) bool {
	return v == v2
}

func (v version) gt(v2 version) bool {
	m1 := v.getMajor()
	m2 := v2.getMajor()
	return m1 > m2
}

func (v version) lt(v2 version) bool {
	m1 := v.getMajor()
	m2 := v2.getMajor()
	return m1 < m2
}

func (v version) ge(v2 version) bool {
	m1 := v.getMajor()
	m2 := v2.getMajor()
	return m1 >= m2
}

func (v version) le(v2 version) bool {
	m1 := v.getMajor()
	m2 := v2.getMajor()
	return m1 <= m2
}

func (v version) getMajor() int {
	if string(v) == "" {
		return 0
	}
	p := strings.Split(string(v), ".")[0]
	b, err := strconv.Atoi(p)
	if err == nil {
		return b
	}
	return 0
}

func defaultQuery() []string {
	return []string{
		"> 0.5%",
		"last 2 versions",
		"Firefox ESR",
		"not dead",
	}
}

func uniq(s ...string) []string {
	var o []string
	m := make(map[string]bool)
	for _, v := range s {
		if m[v] {
			continue
		}
		m[v] = true
		o = append(o, v)
	}
	return o
}

type handler struct {
	match  *regexp.Regexp
	filter func(map[string]data, []string) ([]string, error)
}

func allHandlers() []handler {
	return []handler{
		{
			match:  regexp.MustCompile(`^last\s+(\d+)\s+major versions?$`),
			filter: lastMajorVersions,
		},
		{
			match:  regexp.MustCompile(`^last\s+(\d+)\s+versions?$`),
			filter: lastVersions,
		},
		{
			match:  regexp.MustCompile(`^last\s+(\d+)\s+(\w+)\s+major versions?$`),
			filter: lastMajorVersionsName,
		},
		{
			match:  regexp.MustCompile(`^last\s+(\d+)\s+(\w+)\s+versions?$`),
			filter: lastVersionsName,
		},
		{
			match:  regexp.MustCompile(`^unreleased\s+versions$`),
			filter: unreleased,
		},
		{
			match:  regexp.MustCompile(`^unreleased\s+(\w+)\s+versions?$`),
			filter: unreleasedName,
		},
		{
			match:  regexp.MustCompile(`^last\s+(\d+)\s+years?$`),
			filter: lastYears,
		},
		{
			match:  regexp.MustCompile(`^since (\d+)(?:-(\d+))?(?:-(\d+))?$`),
			filter: sinceDate,
		},
		{
			match:  regexp.MustCompile(`^(>=?|<=?)\s*(\d*\.?\d+)%$`),
			filter: popularitySign,
		},
		{
			match: regexp.MustCompile(`^electron\s+([\d.]+)\s*-\s*([\d.]+)$`),
			filter: func(dataCtx map[string]data, args []string) ([]string, error) {
				from, ok := e2c.Versions[args[0]]
				if !ok {
					return nil, errors.New("unknown version  " + args[0] + "of electron")
				}
				to, ok := e2c.Versions[args[1]]
				if !ok {
					return nil, errors.New("unknown version  " + args[1] + "of electron")
				}
				ff, err := strconv.ParseFloat(from, 64)
				if err != nil {
					return nil, err
				}
				tf, err := strconv.ParseFloat(to, 64)
				if err != nil {
					return nil, err
				}
				var result []string
				for _, v := range e2c.Versions {
					kf, err := strconv.ParseFloat(v, 64)
					if err != nil {
						return nil, err
					}
					if kf >= ff && kf <= tf {
						result = append(result, "chrome "+v)
					}
				}
				sort.Strings(result)
				return result, nil
			},
		},
		{
			match: regexp.MustCompile(`^electron\s*(>=?|<=?)\s*([\d.]+)$`),
			filter: func(dataCtx map[string]data, args []string) ([]string, error) {
				var results []string
				sign := args[0]
				ver := args[1]
				v, err := strconv.ParseFloat(ver, 54)
				if err != nil {
					return nil, err
				}
				for key, value := range e2c.Versions {
					sv, err := strconv.ParseFloat(key, 64)
					if err != nil {
						return nil, err
					}
					switch sign {
					case ">":
						if sv > v {
							results = append(results, "chrome "+value)
						}
					case ">=":
						if sv >= v {
							results = append(results, "chrome "+value)
						}
					case "<":
						if sv < v {
							results = append(results, "chrome "+value)
						}
					case "<=":
						if sv <= v {
							results = append(results, "chrome "+value)
						}
					}
				}
				sort.Strings(results)
				return results, nil
			},
		},
		{
			match: regexp.MustCompile(`^(firefox|ff|fx)\s+esr$`),
			filter: func(_ map[string]data, _ []string) ([]string, error) {
				return []string{"firefox 52", "firefox 60"}, nil
			},
		},
		{
			match: regexp.MustCompile(`(operamini|op_mini)\s+all`),
			filter: func(_ map[string]data, _ []string) ([]string, error) {
				return []string{"op_mini all"}, nil
			},
		},
		{
			match: regexp.MustCompile(`^electron\s+([\d.]+)$`),
			filter: func(_ map[string]data, args []string) ([]string, error) {
				c, ok := e2c.Versions[args[0]]
				if !ok {
					return nil, errors.New("unknown version " + args[0] + "of chrome")
				}
				return []string{"chrome " + c}, nil
			},
		},
		{
			match: regexp.MustCompile(`^defaults$`),
			filter: func(_ map[string]data, _ []string) ([]string, error) {
				return Query(defaultQuery()...)
			},
		},
		{
			match: regexp.MustCompile(`^(\w+)\s*(>=?|<=?)\s*([\d.]+)$`),
			filter: func(dataCtx map[string]data, args []string) ([]string, error) {
				name, sign, ver := args[0], args[1], args[2]
				d, ok := dataCtx[browserName(name)]
				if !ok {
					return nil, fmt.Errorf("unknown browser name :%s", name)
				}
				v, err := strconv.ParseFloat(ver, 64)
				if err != nil {
					return nil, err
				}
				result := filterSlice(d.released, func(s string) bool {
					sv, err := strconv.ParseFloat(s, 64)
					if err != nil {
						return false
					}
					switch sign {
					case ">":
						return sv > v
					case ">=":
						return sv >= v
					case "<":
						return sv < v
					case "<=":
						return sv <= v
					default:
						return false
					}
				})
				return mapNames(d.name, result...), nil
			},
		},
		{
			match: regexp.MustCompile(`^(\w+)\s+(tp|[\d.]+)$`),
			filter: func(dataCtx map[string]data, args []string) ([]string, error) {
				name, ver := args[0], args[1]
				if ver == "tp" {
					ver = "TP"
				}
				d, ok := dataCtx[browserName(name)]
				if !ok {
					return nil, fmt.Errorf("unknown browser name :%s", name)
				}
				alias := normalizeVersion(d, ver)
				if alias != "" {
					ver = alias
				} else {
					if !strings.Contains(ver, ".") {
						alias = ver + ".0"
					} else {
						alias = strings.TrimSuffix(ver, ".0")
					}
					alias = normalizeVersion(d, alias)
					if alias != "" {
						ver = alias
					} else {
						return nil, fmt.Errorf("unknown version %s", ver)
					}
				}
				return []string{d.name + " " + ver}, nil
			},
		},
		{
			match: regexp.MustCompile(`^dead$`),
			filter: func(dataCtx map[string]data, _ []string) ([]string, error) {
				dead := []string{"ie <= 10", "ie_mob <= 10", "bb <= 10", "op_mob <= 12.1"}
				return QueryWith(dataCtx, dead...)
			},
		},
	}
}

func normalizeVersion(d data, ver string) string {
	for _, v := range d.versions {
		if v == ver {
			return v
		}
	}
	a := getAlias()
	if v, ok := a[d.name][ver]; ok {
		return v
	}
	if len(d.versions) == 1 {
		return d.versions[0]
	}
	return ""
}

func popularitySign(dataCtx map[string]data, v []string) ([]string, error) {
	sign := v[0]
	popularity, err := strconv.ParseFloat(v[1], 64)
	if err != nil {
		return nil, err
	}
	var o []string
	for _, name := range agents.Keys() {
		d, ok := dataCtx[name]
		if !ok {
			continue
		}
		var vers []string
		for rel, cov := range d.usage {
			switch sign {
			case ">":
				if cov > popularity {
					vers = append(vers, rel)
				}
			case "<":
				if cov < popularity {
					vers = append(vers, rel)
				}
			case ">=":
				if cov >= popularity {
					vers = append(vers, rel)
				}
			case "<=":
				if cov <= popularity {
					vers = append(vers, rel)
				}
			}
		}
		sort.Strings(vers)
		o = append(o, mapNames(name, vers...)...)
	}
	return o, nil
}

func lastYears(dataCtx map[string]data, v []string) ([]string, error) {
	i, err := strconv.Atoi(v[0])
	if err != nil {
		return nil, err
	}
	now := time.Now()
	year, month, day := now.Date()
	year -= i
	n := time.Date(year, month, day, 0, 0, 0, 0, time.UTC).UnixNano() * 1000
	return filterByYear(dataCtx, n)
}

func sinceDate(dataCtx map[string]data, v []string) ([]string, error) {
	year, err := strconv.Atoi(v[0])
	if err != nil {
		return nil, err
	}
	month := 1
	if v[1] != "" {
		month, err = strconv.Atoi(v[1])
		if err != nil {
			return nil, err
		}
	}
	day := 1
	if v[2] != "" {
		day, err = strconv.Atoi(v[2])
		if err != nil {
			return nil, err
		}
	}
	n := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC).Unix()
	return filterByYear(dataCtx, n)
}

func filterByYear(dataCTx map[string]data, since int64) ([]string, error) {
	var o []string
	for _, name := range agents.Keys() {
		d, ok := dataCTx[name]
		if !ok {
			continue
		}
		var vers []string
		for key, value := range d.releaseDate {
			if value >= since {
				vers = append(vers, key)
			}
		}
		sort.Strings(vers)
		o = append(o, mapNames(name, vers...)...)
	}
	return o, nil
}

func unreleased(dataCtx map[string]data, v []string) ([]string, error) {
	var o []string
	for _, key := range agents.Keys() {
		d, ok := dataCtx[key]
		if !ok {
			continue
		}
		var vers []string
		for _, ver := range d.versions {
			for _, rel := range d.released {
				if rel == ver {
					continue
				}
			}
			vers = append(vers, ver)
		}
		o = append(o, mapNames(key, vers...)...)
	}
	return o, nil
}

func unreleasedName(dataCtx map[string]data, v []string) ([]string, error) {
	name := v[0]
	name = browserName(name)
	d, ok := dataCtx[name]
	if !ok {
		return nil, fmt.Errorf("unknown browser %s", name)
	}
	var vers []string
	for _, ver := range d.versions {
		for _, rel := range d.released {
			if rel == ver {
				continue
			}
		}
		vers = append(vers, ver)
	}
	return mapNames(name, vers...), nil
}

func lastMajorVersions(dataCtx map[string]data, v []string) ([]string, error) {
	var o []string
	ver := 1
	if v[0] != "" {
		i, err := strconv.Atoi(v[0])
		if err != nil {
			return nil, err
		}
		ver = i
	}
	for _, k := range agents.Keys() {
		d, ok := dataCtx[k]
		if !ok {
			return []string{}, nil
		}
		i, err := getMajorVersions(d.released, ver)
		if err != nil {
			return nil, err
		}
		o = append(o, mapNames(d.name, i...)...)
	}
	return o, nil
}
func lastMajorVersionsName(dataCtx map[string]data, v []string) ([]string, error) {
	ver, err := strconv.Atoi(v[0])
	if err != nil {
		return nil, err
	}
	name := v[1]
	name = browserName(name)
	d, ok := dataCtx[name]
	if !ok {
		return nil, fmt.Errorf("unknown browser %s", name)
	}
	i, err := getMajorVersions(d.released, ver)
	if err != nil {
		return nil, err
	}
	return mapNames(name, i...), nil
}

func lastVersions(dataCtx map[string]data, v []string) ([]string, error) {
	ver, err := strconv.Atoi(v[0])
	if err != nil {
		return nil, err
	}
	var o []string
	for _, k := range agents.Keys() {
		d, ok := dataCtx[k]
		if !ok {
			continue
		}
		if len(d.released) > ver {
			idx := len(d.released) - ver
			o = append(o, mapNames(d.name, d.released[idx:]...)...)
		} else {
			o = append(o, mapNames(d.name, d.released...)...)
		}
	}
	return o, nil
}

func lastVersionsName(dataCtx map[string]data, v []string) ([]string, error) {
	ver, err := strconv.Atoi(v[0])
	if err != nil {
		return nil, err
	}
	name := v[1]
	name = browserName(name)
	d, ok := dataCtx[name]
	if !ok {
		return nil, fmt.Errorf("unknown browser %s", name)
	}
	if len(d.released) > ver {
		idx := len(d.released) - ver
		return mapNames(d.name, d.released[idx:]...), nil
	}
	return mapNames(d.name, d.released...), nil
}

func mapNames(base string, s ...string) (o []string) {
	for _, v := range s {
		o = append(o, base+" "+v)
	}
	return
}

func getMajorVersions(released []string, number int) ([]string, error) {
	if len(released) == 0 {
		return []string{}, nil
	}
	min := version(released[0]).getMajor() - number + 1
	var o []string
	for k, v := range released {
		if k == 0 {
			continue
		}
		if version(v).getMajor() > min {
			o = append(o, v)
		}
	}
	return o, nil
}

type data struct {
	name        string
	versions    []string
	released    []string
	releaseDate map[string]int64
	usage       map[string]float64
}

func getData() map[string]data {
	m := make(map[string]data)
	for k, v := range agents.New() {
		ve := normalize(v.Versions...)
		d := data{
			name:        k,
			versions:    ve,
			releaseDate: v.ReleaseDate,
			usage:       v.UsageGlobal,
		}
		if len(ve) > 2 {
			d.released = ve[len(ve)-2:]
		}
		m[k] = d
	}
	return m
}

func getAlias() map[string]map[string]string {
	o := make(map[string]map[string]string)
	for k, v := range getData() {
		m := make(map[string]string)
		for _, ver := range v.versions {
			if ver != "" {
				if strings.Contains(ver, "-") {
					for _, p := range strings.Split(ver, "-") {
						m[p] = ver
					}
				}
			}
		}
		o[k] = m
	}
	return o
}

func normalize(s ...string) []string {
	var o []string
	for _, v := range s {
		if v != "" {
			o = append(o, v)
		}
	}
	return o
}

func Query(s ...string) ([]string, error) {
	return QueryWith(getData(), s...)
}

func QueryWith(dataCtx map[string]data, s ...string) ([]string, error) {
	h := allHandlers()
	var o []string
	for _, v := range s {
		v = strings.ToLower(v)
		exclude := false
		if strings.HasPrefix(v, "not ") {
			exclude = true
			v = strings.TrimSpace(v[4:])
		}
		match := false
		for _, c := range h {
			if match {
				break
			}
			if c.match.MatchString(v) {
				if !match {
					match = true
				}
				i, err := c.filter(dataCtx, c.match.FindStringSubmatch(v)[1:])
				if err != nil {
					return nil, err
				}
				if exclude {
					o = filterSlice(o, func(sv string) bool {
						for _, val := range i {
							if val == sv {
								return true
							}
						}
						return false
					})
				} else {
					o = append(o, i...)
				}
				continue
			}
		}
		if !match {
			return nil, fmt.Errorf("unknown query %s", v)
		}
	}
	o = uniq(o...)
	sortResults(o)
	return o, nil
}

func filterSlice(s []string, fn func(string) bool) []string {
	var o []string
	for _, v := range s {
		if fn(v) {
			o = append(o, v)
		}
	}
	return o
}

func sortResults(results []string) {
	sort.Slice(results, func(i, j int) bool {
		pi := strings.Split(results[i], " ")
		pj := strings.Split(results[i], " ")
		return pi[0] < pj[0]
	})
}
