package libraries

import (
	"context"
	"database/sql"
	"encoding/json"
	"sort"
	"strings"

	"gestor-documental/internal/domain"
)

type CaseStatisticsGroup struct {
	Key              string   `json:"key"`
	Label            string   `json:"label"`
	Total            int      `json:"total"`
	Evaluable        int      `json:"evaluable"`
	Complete         int      `json:"complete"`
	Incomplete       int      `json:"incomplete"`
	WithoutDocuments int      `json:"without_documents"`
	WithoutTemplate  int      `json:"without_template"`
	Required         int      `json:"required"`
	Fulfilled        int      `json:"fulfilled"`
	Unavailable      int      `json:"unavailable"`
	CompletePercent  *float64 `json:"complete_percent"`
	ProgressPercent  *float64 `json:"progress_percent"`
}
type MissingRequirement struct {
	Name         string `json:"name"`
	Cases        int    `json:"cases"`
	MissingUnits int    `json:"missing_units"`
}
type CaseStatisticsResult struct {
	Totals     CaseStatisticsGroup   `json:"totals"`
	ByExercise []CaseStatisticsGroup `json:"by_exercise"`
	ByTemplate []CaseStatisticsGroup `json:"by_template"`
	Missing    []MissingRequirement  `json:"missing_requirements"`
}

func (s *Service) CaseExercises(ctx context.Context, p domain.Principal, library string) ([]string, error) {
	result := []string{}
	if err := s.Read(ctx, p, library, "documents.read"); err != nil {
		return result, err
	}
	rows, err := s.Database.Reader.QueryContext(ctx, "SELECT DISTINCT exercise FROM cases WHERE library_id=? AND deleted_at IS NULL AND exercise IS NOT NULL AND exercise<>'' ORDER BY exercise DESC", library)
	if err != nil {
		return result, err
	}
	defer rows.Close()
	for rows.Next() {
		var year string
		if err := rows.Scan(&year); err != nil {
			return result, err
		}
		result = append(result, year)
	}
	return result, rows.Err()
}

// The same availability and approval rules used by the case detail define
// valid evidence here. All relationships are read on demand; no periodic
// statistics scheduler is introduced.
const caseStatsValidDocs = `valid_docs AS (
 SELECT d.case_id,d.document_type_id,count(*) AS valid_count
 FROM documents d JOIN physical_files f ON f.id=d.physical_file_id
 LEFT JOIN physical_file_locations loc ON loc.id=f.primary_location_id
 LEFT JOIN storage_roots sr ON sr.id=loc.root_id
 WHERE d.library_id=? AND d.deleted_at IS NULL AND d.case_id IS NOT NULL
 AND d.approval_status='approved' AND f.availability='available'
 AND coalesce(sr.status,'')<>'inaccessible'
 AND (f.storage_source='linked' OR f.integrity_status='verified')
 AND NOT EXISTS(SELECT 1 FROM upload_items u WHERE u.document_id=d.id AND u.status IN ('staged','retained'))
 GROUP BY d.case_id,d.document_type_id
)`

func caseStatsAdd(group *CaseStatisticsGroup, expected, fulfilled, documents, unavailable int) {
	group.Total++
	group.Unavailable += unavailable
	if documents == 0 {
		group.WithoutDocuments++
	}
	if expected <= 0 {
		group.WithoutTemplate++
		return
	}
	group.Evaluable++
	group.Required += expected
	group.Fulfilled += fulfilled
	if fulfilled >= expected {
		group.Complete++
	} else {
		group.Incomplete++
	}
}
func caseStatsPercent(group *CaseStatisticsGroup) {
	if group.Evaluable > 0 {
		value := 100 * float64(group.Complete) / float64(group.Evaluable)
		group.CompletePercent = &value
	}
	if group.Required > 0 {
		value := 100 * float64(group.Fulfilled) / float64(group.Required)
		group.ProgressPercent = &value
	}
}
func (s *Service) CaseStatistics(ctx context.Context, p domain.Principal, library, exercise string) (CaseStatisticsResult, error) {
	result := CaseStatisticsResult{Totals: CaseStatisticsGroup{Key: "all", Label: "Todos los expedientes"}, ByExercise: []CaseStatisticsGroup{}, ByTemplate: []CaseStatisticsGroup{}, Missing: []MissingRequirement{}}
	if err := s.Read(ctx, p, library, "documents.read"); err != nil {
		return result, err
	}
	if exercise != "" && (len(exercise) != 4 || strings.IndexFunc(exercise, func(r rune) bool { return r < '0' || r > '9' }) != -1) {
		return result, invalid("Ejercicio inválido.")
	}
	tx, err := s.Database.Reader.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return result, err
	}
	defer tx.Rollback()
	query := `WITH ` + caseStatsValidDocs + `, requirements AS (
 SELECT r.case_id,sum(r.required_count) expected,sum(least(r.required_count,coalesce(v.valid_count,0))) fulfilled,
 json_agg(json_build_object('type',r.document_type_id,'name',t.name,'missing',greatest(r.required_count-coalesce(v.valid_count,0),0))) missing_json
 FROM case_requirements r JOIN document_types t ON t.id=r.document_type_id
 LEFT JOIN valid_docs v ON v.case_id=r.case_id AND v.document_type_id=r.document_type_id
 WHERE r.library_id=? AND r.mandatory=1 GROUP BY r.case_id
), doc_counts AS (
 SELECT d.case_id,count(*) AS total,sum(CASE WHEN f.availability<>'available' OR coalesce(sr.status,'')='inaccessible' THEN 1 ELSE 0 END) unavailable
 FROM documents d JOIN physical_files f ON f.id=d.physical_file_id
 LEFT JOIN physical_file_locations loc ON loc.id=f.primary_location_id LEFT JOIN storage_roots sr ON sr.id=loc.root_id
 WHERE d.library_id=? AND d.case_id IS NOT NULL AND d.deleted_at IS NULL AND d.approval_status<>'cancelled'
 AND NOT EXISTS(SELECT 1 FROM upload_items u WHERE u.document_id=d.id AND u.status IN ('staged','retained'))
 GROUP BY d.case_id
)
 SELECT coalesce(c.exercise,''),coalesce(t.id,''),coalesce(t.name,'Sin plantilla'),
 coalesce(r.expected,0),coalesce(r.fulfilled,0),coalesce(dc.total,0),coalesce(dc.unavailable,0),coalesce(r.missing_json,'[]')
 FROM cases c LEFT JOIN requirements r ON r.case_id=c.id LEFT JOIN doc_counts dc ON dc.case_id=c.id
 LEFT JOIN template_versions tv ON tv.id=c.template_version_id LEFT JOIN templates t ON t.id=tv.template_id
 WHERE c.library_id=? AND c.deleted_at IS NULL AND (?='' OR c.exercise=?)`
	rows, err := tx.QueryContext(ctx, query, library, library, library, library, exercise, exercise)
	if err != nil {
		return result, err
	}
	byExercise, byTemplate := map[string]*CaseStatisticsGroup{}, map[string]*CaseStatisticsGroup{}
	missingByType := map[string]*MissingRequirement{}
	for rows.Next() {
		var year, templateID, templateName string
		var missingJSON string
		var expected, fulfilled, documents, unavailable int
		if err = rows.Scan(&year, &templateID, &templateName, &expected, &fulfilled, &documents, &unavailable, &missingJSON); err != nil {
			rows.Close()
			return result, err
		}
		var requirements []struct {
			Type    string `json:"type"`
			Name    string `json:"name"`
			Missing int    `json:"missing"`
		}
		if err = json.Unmarshal([]byte(missingJSON), &requirements); err != nil {
			rows.Close()
			return result, err
		}
		for _, requirement := range requirements {
			if requirement.Missing == 0 {
				continue
			}
			item := missingByType[requirement.Type]
			if item == nil {
				item = &MissingRequirement{Name: requirement.Name}
				missingByType[requirement.Type] = item
			}
			item.Cases++
			item.MissingUnits += requirement.Missing
		}
		caseStatsAdd(&result.Totals, expected, fulfilled, documents, unavailable)
		if _, ok := byExercise[year]; !ok {
			label := year
			if label == "" {
				label = "Sin ejercicio"
			}
			byExercise[year] = &CaseStatisticsGroup{Key: year, Label: label}
		}
		caseStatsAdd(byExercise[year], expected, fulfilled, documents, unavailable)
		if _, ok := byTemplate[templateID]; !ok {
			byTemplate[templateID] = &CaseStatisticsGroup{Key: templateID, Label: templateName}
		}
		caseStatsAdd(byTemplate[templateID], expected, fulfilled, documents, unavailable)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return result, err
	}
	for _, g := range byExercise {
		caseStatsPercent(g)
		result.ByExercise = append(result.ByExercise, *g)
	}
	for _, g := range byTemplate {
		caseStatsPercent(g)
		result.ByTemplate = append(result.ByTemplate, *g)
	}
	caseStatsPercent(&result.Totals)
	// Each mandatory requirement contributes once per incomplete case.
	for _, item := range missingByType {
		result.Missing = append(result.Missing, *item)
	}
	sort.Slice(result.Missing, func(i, j int) bool {
		if result.Missing[i].Cases != result.Missing[j].Cases {
			return result.Missing[i].Cases > result.Missing[j].Cases
		}
		return result.Missing[i].Name < result.Missing[j].Name
	})
	if len(result.Missing) > 10 {
		result.Missing = result.Missing[:10]
	}
	return result, tx.Commit()
}
