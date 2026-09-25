package libraries

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"gestor-documental/internal/domain"
	"gestor-documental/internal/licensing"
	"gestor-documental/internal/storage"
)

type Settings struct {
	ReviewManaged    bool   `json:"review_managed"`
	ReviewLinked     bool   `json:"review_linked"`
	RetentionDays    int    `json:"retention_days"`
	IdentifierLabel  string `json:"identifier_label"`
	ExerciseEnabled  bool   `json:"exercise_enabled"`
	CasesEnabled     bool   `json:"cases_enabled"`
	StructurePattern string `json:"structure_pattern"`
	FilenamePattern  string `json:"filename_pattern"`
	FilenamePrefix   string `json:"filename_prefix"`
	ManagedRootID    string `json:"default_managed_root_id"`
}

func defaultSettings() Settings {
	return Settings{IdentifierLabel: "Expediente", CasesEnabled: true, StructurePattern: "{identificador}/{categoria}", FilenamePattern: "original"}
}
func settingsFor(ctx context.Context, query storage.Querier, libraryID string) (Settings, string, error) {
	var settings Settings
	var content, mode string
	err := query.QueryRowContext(ctx, "SELECT settings_json,mode FROM libraries WHERE id=? AND disabled_at IS NULL", libraryID).Scan(&content, &mode)
	if err == sql.ErrNoRows {
		return settings, mode, notFound()
	}
	if err != nil {
		return settings, mode, err
	}
	err = json.Unmarshal([]byte(content), &settings)
	return settings, mode, err
}
func modeCapabilities(mode string) []string {
	switch mode {
	case "linked":
		return []string{"linked_libraries"}
	case "managed":
		return []string{"managed_libraries"}
	case "hybrid":
		return []string{"linked_libraries", "managed_libraries"}
	}
	return nil
}
func requireOrganization(ctx context.Context, query storage.Querier, libraryID string) (Settings, error) {
	settings, mode, err := settingsFor(ctx, query, libraryID)
	if err != nil {
		return settings, err
	}
	if mode == "linked" || !settings.CasesEnabled {
		return settings, invalid("Habilita expedientes en una biblioteca administrada o híbrida.")
	}
	return settings, licensing.CheckFeatures(append(modeCapabilities(mode), "expedientes")...)
}
func validSettings(settings Settings) error {
	if settings.RetentionDays < 0 || settings.RetentionDays > 3650 {
		return invalid("La retención debe ser de 0 (conservar) a 3650 días.")
	}
	if strings.TrimSpace(settings.IdentifierLabel) == "" || utf8.RuneCountInString(settings.IdentifierLabel) > 60 || utf8.RuneCountInString(settings.FilenamePrefix) > 40 {
		return invalid("Etiqueta o prefijo inválido.")
	}
	switch settings.StructurePattern {
	case "{identificador}/{categoria}", "{categoria}/{identificador}", "{identificador}", "{categoria}":
	case "{ejercicio}/{identificador}/{categoria}":
		if !settings.ExerciseEnabled {
			return invalid("La estructura requiere habilitar ejercicio.")
		}
	default:
		return invalid("Selecciona una estructura disponible.")
	}
	if !settings.CasesEnabled && strings.Contains(settings.StructurePattern, "{identificador}") {
		return invalid("La estructura requiere expedientes; selecciona solo categoría.")
	}
	switch settings.FilenamePattern {
	case "original", "{tipo_documento}_{consecutivo}", "{prefijo}_{identificador}_{tipo_documento}_{consecutivo}", "{ejercicio}_{tipo_documento}_{consecutivo}":
	default:
		return invalid("Selecciona un formato de nombre disponible.")
	}
	if strings.Contains(settings.FilenamePattern, "{ejercicio}") && !settings.ExerciseEnabled {
		return invalid("El nombre requiere habilitar ejercicio.")
	}
	if strings.Contains(settings.FilenamePattern, "{identificador}") && !settings.CasesEnabled {
		return invalid("El nombre requiere expedientes.")
	}
	return nil
}
func (service *Service) updateSettings(ctx context.Context, transaction *sql.Tx, libraryID, mode string, settings *Settings) error {
	previous, oldMode, err := settingsFor(ctx, transaction, libraryID)
	if err != nil {
		return err
	}
	if mode == "" {
		mode = oldMode
	}
	if len(modeCapabilities(mode)) == 0 || (oldMode != mode && mode != "hybrid") {
		return invalid("Solo se permite ampliar una biblioteca vinculada o administrada a híbrida.")
	}
	if err = licensing.CheckFeatures(modeCapabilities(mode)...); err != nil {
		return err
	}
	if settings == nil {
		settings = &previous
	}
	if settings.ReviewManaged || settings.ReviewLinked {
		if err = licensing.CheckFeatures("review_workflow"); err != nil {
			return err
		}
	}
	if err = validSettings(*settings); err != nil {
		return err
	}
	if previous.CasesEnabled && !settings.CasesEnabled {
		var count int
		if err = transaction.QueryRowContext(ctx, "SELECT count(*) FROM cases WHERE library_id=?", libraryID).Scan(&count); err != nil {
			return err
		}
		if count > 0 {
			return invalid("No puedes deshabilitar expedientes mientras existan registros.")
		}
	}
	if previous.ExerciseEnabled && !settings.ExerciseEnabled {
		var count int
		if err = transaction.QueryRowContext(ctx, "SELECT count(*) FROM cases WHERE library_id=? AND exercise IS NOT NULL", libraryID).Scan(&count); err != nil {
			return err
		}
		if count > 0 {
			return invalid("Conserva ejercicio mientras existan expedientes que lo utilizan.")
		}
	}
	if settings.ManagedRootID != "" {
		var count int
		if err = transaction.QueryRowContext(ctx, "SELECT count(*) FROM storage_roots WHERE id=? AND library_id=? AND storage_source='managed' AND status='active'", settings.ManagedRootID, libraryID).Scan(&count); err != nil {
			return err
		}
		if count != 1 || mode == "linked" {
			return invalid("Selecciona un destino administrado activo de esta biblioteca.")
		}
	}
	_, err = transaction.ExecContext(ctx, "UPDATE libraries SET mode=?,settings_json=? WHERE id=?", mode, encode(settings), libraryID)
	return err
}
func configurationSnapshot(ctx context.Context, transaction *sql.Tx, libraryID, userID string) error {
	_, err := transaction.ExecContext(ctx, "INSERT INTO library_configuration_versions SELECT id,current_configuration_revision,mode,settings_json,?,? FROM libraries WHERE id=?", userID, now(), libraryID)
	return err
}

// Each token becomes one safe component. The stable opaque suffix prevents
// collisions without encoding an internal absolute path or renaming older files.
func safeComponent(value string) string {
	var clean strings.Builder
	for _, character := range strings.TrimSpace(value) {
		if unicode.IsLetter(character) || unicode.IsDigit(character) || character == '-' || character == '_' || character == '.' {
			clean.WriteRune(character)
		} else {
			clean.WriteByte('_')
		}
	}
	characters := []rune(strings.Trim(clean.String(), ". _"))
	if len(characters) > 60 {
		characters = characters[:60]
	}
	result := string(characters)
	if result == "" {
		result = "Documento"
	}
	first := strings.ToUpper(strings.Split(result, ".")[0])
	if first == "CON" || first == "PRN" || first == "AUX" || first == "NUL" || len(first) == 4 && (strings.HasPrefix(first, "COM") || strings.HasPrefix(first, "LPT")) && first[3] >= '1' && first[3] <= '9' {
		result = "_" + result
	}
	return result
}

type NamingInput struct {
	Identifier       string `json:"identifier"`
	Exercise         string `json:"exercise"`
	Category         string `json:"category"`
	DocumentType     string `json:"document_type"`
	OriginalFilename string `json:"original_filename"`
}

func renderNaming(settings Settings, input NamingInput, stableID string) string {
	return renderNamingSequence(settings, input, stableID, 1)
}
func renderNamingSequence(settings Settings, input NamingInput, stableID string, sequence int64) string {
	tokens := strings.NewReplacer("{identificador}", safeComponent(input.Identifier), "{ejercicio}", safeComponent(input.Exercise), "{categoria}", safeComponent(input.Category), "{tipo_documento}", safeComponent(input.DocumentType), "{prefijo}", safeComponent(settings.FilenamePrefix), "{consecutivo}", fmt.Sprintf("%04d", sequence))
	filename := tokens.Replace(settings.FilenamePattern)
	if settings.FilenamePattern == "original" {
		filename = safeComponent(strings.TrimSuffix(input.OriginalFilename, ".pdf"))
	}
	return tokens.Replace(settings.StructurePattern) + "/" + safeComponent(filename) + "_" + stableID + ".pdf"
}
func (service *Service) NamingPreview(ctx context.Context, principal domain.Principal, libraryID string, input NamingInput) (map[string]string, error) {
	if err := service.Read(ctx, principal, libraryID, "libraries.configure"); err != nil {
		return nil, err
	}
	settings, _, err := settingsFor(ctx, service.Database.Reader, libraryID)
	if err != nil {
		return nil, err
	}
	for _, value := range []string{input.Identifier, input.Exercise, input.Category, input.DocumentType, input.OriginalFilename} {
		if utf8.RuneCountInString(value) > 250 {
			return nil, invalid("Valor demasiado largo.")
		}
	}
	return map[string]string{"relative_path": renderNaming(settings, input, "ID_ESTABLE"), "note": fmt.Sprintf("Los cambios solo afectan incorporaciones futuras. %s es la etiqueta visible.", settings.IdentifierLabel)}, nil
}
