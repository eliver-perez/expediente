package libraries

import (
	"context"
	"database/sql"
	"strings"
	"unicode/utf8"

	"gestor-documental/internal/domain"
)

type CatalogEntry struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	CategoryID    string `json:"category_id"`
	Multiple      bool   `json:"allows_multiple"`
	TitleRequired bool   `json:"requires_descriptive_title"`
	Archived      bool   `json:"archived"`
	Revision      int64  `json:"revision"`
}

func validName(name string) bool {
	return strings.TrimSpace(name) != "" && utf8.RuneCountInString(name) <= 160
}
func conflict() error {
	return domain.Failure("VERSION_CONFLICT", "El registro cambió. Actualiza la página.", 412)
}
func (service *Service) Catalog(ctx context.Context, principal domain.Principal, libraryID, kind string) ([]CatalogEntry, error) {
	if err := service.Read(ctx, principal, libraryID, "documents.read"); err != nil {
		return nil, err
	}
	result := []CatalogEntry{}
	statement := "SELECT id,name,'',0,0,archived_at IS NOT NULL,revision FROM categories WHERE library_id=? ORDER BY name,id"
	if kind == "document-types" {
		statement = "SELECT id,name,category_id,allows_multiple,requires_descriptive_title,archived_at IS NOT NULL,revision FROM document_types WHERE library_id=? ORDER BY name,id"
	}
	rows, err := service.Database.Reader.QueryContext(ctx, statement, libraryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var entry CatalogEntry
		if err = rows.Scan(&entry.ID, &entry.Name, &entry.CategoryID, &entry.Multiple, &entry.TitleRequired, &entry.Archived, &entry.Revision); err != nil {
			return nil, err
		}
		result = append(result, entry)
	}
	return result, rows.Err()
}
func (service *Service) SaveCatalog(ctx context.Context, principal domain.Principal, libraryID, kind, identifier string, input CatalogEntry, revision int64, metadata domain.RequestMetadata) (string, error) {
	input.Name = strings.TrimSpace(input.Name)
	if !validName(input.Name) || (kind != "categories" && kind != "document-types") {
		return "", invalid("Nombre o catálogo inválido.")
	}
	creating := identifier == ""
	if creating {
		identifier = domain.NewID()
	}
	err := service.write(ctx, principal, libraryID, "catalogs.manage", func(transaction *sql.Tx, current domain.Principal) error {
		if err := requireManaged(ctx, transaction, libraryID); err != nil {
			return err
		}
		var previousRevision int64
		if !creating {
			table := "categories"
			if kind == "document-types" {
				table = "document_types"
			}
			if err := transaction.QueryRowContext(ctx, "SELECT revision FROM "+table+" WHERE id=? AND library_id=?", identifier, libraryID).Scan(&previousRevision); err == sql.ErrNoRows {
				return notFound()
			} else if err != nil {
				return err
			}
			if previousRevision != revision {
				return conflict()
			}
		}
		archived := any(nil)
		if input.Archived {
			archived = now()
		}
		if kind == "categories" {
			if creating {
				if _, err := transaction.ExecContext(ctx, "INSERT INTO categories(id,library_id,name) VALUES(?,?,?)", identifier, libraryID, input.Name); err != nil {
					return err
				}
			} else {
				if _, err := transaction.ExecContext(ctx, "UPDATE categories SET name=?,archived_at=?,revision=revision+1 WHERE id=?", input.Name, archived, identifier); err != nil {
					return err
				}
			}
		} else {
			var active int
			if err := transaction.QueryRowContext(ctx, "SELECT count(*) FROM categories WHERE id=? AND library_id=? AND archived_at IS NULL", input.CategoryID, libraryID).Scan(&active); err != nil {
				return err
			}
			if active != 1 {
				return invalid("Selecciona una categoría activa de esta biblioteca.")
			}
			if creating {
				if _, err := transaction.ExecContext(ctx, "INSERT INTO document_types(id,library_id,category_id,name,allows_multiple,requires_descriptive_title) VALUES(?,?,?,?,?,?)", identifier, libraryID, input.CategoryID, input.Name, input.Multiple, input.TitleRequired); err != nil {
					return err
				}
			} else {
				var category string
				if err := transaction.QueryRowContext(ctx, "SELECT category_id FROM document_types WHERE id=?", identifier).Scan(&category); err != nil {
					return err
				}
				if category != input.CategoryID {
					return invalid("La categoría de un tipo existente se conserva para proteger sus referencias.")
				}
				if _, err := transaction.ExecContext(ctx, "UPDATE document_types SET name=?,allows_multiple=?,requires_descriptive_title=?,archived_at=?,revision=revision+1 WHERE id=?", input.Name, input.Multiple, input.TitleRequired, archived, identifier); err != nil {
					return err
				}
			}
		}
		return record(ctx, transaction, current, metadata, "catalog.saved", libraryID, "", map[string]any{"catalog": kind, "id": identifier, "name": input.Name, "archived": input.Archived})
	})
	return identifier, err
}

type RequirementInput struct {
	CategoryID string `json:"category_id"`
	TypeID     string `json:"document_type_id"`
	Count      int    `json:"required_count"`
	Mandatory  bool   `json:"mandatory"`
}
type TemplateRequirement struct {
	RequirementInput
	ID       string `json:"id"`
	Multiple bool   `json:"allows_multiple"`
}
type TemplateVersion struct {
	ID           string                `json:"id"`
	Revision     int64                 `json:"revision"`
	Requirements []TemplateRequirement `json:"requirements"`
}
type Template struct {
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	Versions []TemplateVersion `json:"versions"`
}

func (service *Service) Templates(ctx context.Context, principal domain.Principal, libraryID string) ([]Template, error) {
	if err := service.Read(ctx, principal, libraryID, "documents.read"); err != nil {
		return nil, err
	}
	result := []Template{}
	rows, err := service.Database.Reader.QueryContext(ctx, "SELECT id,name FROM templates WHERE library_id=? AND archived_at IS NULL ORDER BY name,id", libraryID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var item Template
		if err = rows.Scan(&item.ID, &item.Name); err != nil {
			rows.Close()
			return nil, err
		}
		item.Versions = []TemplateVersion{}
		result = append(result, item)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	for index := range result {
		rows, err = service.Database.Reader.QueryContext(ctx, "SELECT id,revision FROM template_versions WHERE template_id=? ORDER BY revision DESC", result[index].ID)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var version TemplateVersion
			if err = rows.Scan(&version.ID, &version.Revision); err != nil {
				rows.Close()
				return nil, err
			}
			version.Requirements = []TemplateRequirement{}
			result[index].Versions = append(result[index].Versions, version)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return nil, err
		}
		for position := range result[index].Versions {
			version := &result[index].Versions[position]
			rows, err = service.Database.Reader.QueryContext(ctx, "SELECT id,category_id,document_type_id,required_count,mandatory,allows_multiple FROM template_requirements WHERE template_version_id=? ORDER BY id", version.ID)
			if err != nil {
				return nil, err
			}
			for rows.Next() {
				var requirement TemplateRequirement
				if err = rows.Scan(&requirement.ID, &requirement.CategoryID, &requirement.TypeID, &requirement.Count, &requirement.Mandatory, &requirement.Multiple); err != nil {
					rows.Close()
					return nil, err
				}
				version.Requirements = append(version.Requirements, requirement)
			}
			err = rows.Err()
			rows.Close()
			if err != nil {
				return nil, err
			}
		}
	}
	return result, nil
}
func (service *Service) SaveTemplate(ctx context.Context, principal domain.Principal, libraryID, templateID, name string, requirements []RequirementInput, revision int64, metadata domain.RequestMetadata) (string, error) {
	creating := templateID == ""
	if creating && !validName(name) || len(requirements) > 200 {
		return "", invalid("Nombre inválido o más de 200 requerimientos.")
	}
	if creating {
		templateID = domain.NewID()
	}
	versionID := domain.NewID()
	err := service.write(ctx, principal, libraryID, "templates.manage", func(transaction *sql.Tx, current domain.Principal) error {
		if _, err := requireOrganization(ctx, transaction, libraryID); err != nil {
			return err
		}
		var latest int64
		if creating {
			if _, err := transaction.ExecContext(ctx, "INSERT INTO templates(id,library_id,name) VALUES(?,?,?)", templateID, libraryID, strings.TrimSpace(name)); err != nil {
				return err
			}
		} else {
			if err := transaction.QueryRowContext(ctx, "SELECT max(v.revision) FROM template_versions v JOIN templates t ON t.id=v.template_id WHERE t.id=? AND t.library_id=? AND t.archived_at IS NULL GROUP BY t.id", templateID, libraryID).Scan(&latest); err == sql.ErrNoRows {
				return notFound()
			} else if err != nil {
				return err
			}
			if latest != revision {
				return conflict()
			}
		}
		if _, err := transaction.ExecContext(ctx, "INSERT INTO template_versions VALUES(?,?,?,?,?)", versionID, templateID, libraryID, latest+1, now()); err != nil {
			return err
		}
		seen := map[string]bool{}
		for _, requirement := range requirements {
			if seen[requirement.TypeID] || requirement.Count < 1 || requirement.Count > 1000 {
				return invalid("No repitas tipos; cada cantidad debe estar entre 1 y 1000.")
			}
			seen[requirement.TypeID] = true
			var multiple bool
			if err := transaction.QueryRowContext(ctx, "SELECT t.allows_multiple FROM document_types t JOIN categories c ON c.id=t.category_id WHERE t.id=? AND t.category_id=? AND t.library_id=? AND t.archived_at IS NULL AND c.archived_at IS NULL", requirement.TypeID, requirement.CategoryID, libraryID).Scan(&multiple); err == sql.ErrNoRows {
				return invalid("La plantilla contiene un tipo ajeno, inexistente o archivado.")
			} else if err != nil {
				return err
			}
			if !multiple && requirement.Count != 1 {
				return invalid("Un tipo de archivo único requiere cantidad uno.")
			}
			if _, err := transaction.ExecContext(ctx, "INSERT INTO template_requirements VALUES(?,?,?,?,?,?,?,?)", domain.NewID(), versionID, libraryID, requirement.CategoryID, requirement.TypeID, requirement.Count, requirement.Mandatory, multiple); err != nil {
				return err
			}
		}
		return record(ctx, transaction, current, metadata, "template.version_created", libraryID, "", map[string]any{"template_id": templateID, "version_id": versionID, "revision": latest + 1})
	})
	return versionID, err
}
