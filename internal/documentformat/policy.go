package documentformat

import (
	"gestor-documental/internal/domain"
	"slices"
)

func domainInvalid(message string) error { return domain.Failure("INVALID_FILE_POLICY", message, 422) }

type Policy struct {
	Store          []string `json:"store_formats"`
	Index          []string `json:"index_formats"`
	MaximumFileMB  int      `json:"maximum_file_mb"`
	MaximumIndexMB int      `json:"maximum_index_mb"`
	Unknown        string   `json:"unknown_policy"`
	NonIndexable   string   `json:"non_indexable_policy"`
}

// Nil means inherit; an empty list is an intentional ban. Independent fields
// permit a library to override formats while continuing to inherit its limits.
type Overrides struct {
	Store          *[]string `json:"store_formats"`
	Index          *[]string `json:"index_formats"`
	MaximumFileMB  *int      `json:"maximum_file_mb"`
	MaximumIndexMB *int      `json:"maximum_index_mb"`
	Unknown        *string   `json:"unknown_policy"`
	NonIndexable   *string   `json:"non_indexable_policy"`
}

func Default(maximum int) Policy {
	if maximum <= 0 {
		maximum = 256
	}
	return Policy{[]string{"pdf", "docx", "xlsx", "txt", "csv"}, []string{"pdf"}, maximum, maximum, "reject", "store"}
}

func Resolve(base Policy, override Overrides) Policy {
	if override.Store != nil {
		base.Store = slices.Clone(*override.Store)
	}
	if override.Index != nil {
		base.Index = slices.Clone(*override.Index)
	}
	if override.MaximumFileMB != nil {
		base.MaximumFileMB = *override.MaximumFileMB
	}
	if override.MaximumIndexMB != nil {
		base.MaximumIndexMB = *override.MaximumIndexMB
	}
	if override.Unknown != nil {
		base.Unknown = *override.Unknown
	}
	if override.NonIndexable != nil {
		base.NonIndexable = *override.NonIndexable
	}
	if override.Index == nil {
		base.Index = slices.DeleteFunc(slices.Clone(base.Index), func(id string) bool { return !slices.Contains(base.Store, id) })
	}
	if override.MaximumIndexMB == nil {
		base.MaximumIndexMB = min(base.MaximumIndexMB, base.MaximumFileMB)
	}
	return base
}

func (policy Policy) Validate() error {
	if policy.Store == nil || policy.Index == nil {
		return domainInvalid("Indica las listas de formatos de almacenamiento e indexación; pueden estar vacías.")
	}
	for _, formats := range [][]string{policy.Store, policy.Index} {
		seen := map[string]bool{}
		for _, id := range formats {
			if _, valid := Lookup(id); !valid || seen[id] {
				return domainInvalid("Selecciona únicamente formatos admitidos, sin duplicados.")
			}
			seen[id] = true
		}
	}
	for _, id := range policy.Index {
		if !slices.Contains(policy.Store, id) {
			return domainInvalid("Todo formato indexable debe estar permitido para almacenar.")
		}
	}
	if policy.MaximumFileMB < 1 || policy.MaximumFileMB > 4096 || policy.MaximumIndexMB < 1 || policy.MaximumIndexMB > policy.MaximumFileMB {
		return domainInvalid("El tamaño máximo debe ser de 1 a 4096 MiB; el límite de indexación no puede superar el de almacenamiento.")
	}
	if policy.Unknown != "reject" && policy.Unknown != "store_text" {
		return domainInvalid("Selecciona una política válida para formatos desconocidos.")
	}
	if policy.NonIndexable != "store" && policy.NonIndexable != "reject" {
		return domainInvalid("Selecciona conservar o rechazar documentos no indexables.")
	}
	return nil
}

func (policy Policy) IndexReason(detection Detection, size int64) string {
	if detection.Mismatch {
		return "unknown_extension"
	}
	if !slices.Contains(policy.Store, detection.Format) || !slices.Contains(policy.Index, detection.Format) {
		return "format_not_indexed"
	}
	if size > int64(policy.MaximumIndexMB)<<20 {
		return "index_size_limit"
	}
	format, exists := Lookup(detection.Format)
	if !exists || !format.ExtractorAvailable {
		return "extractor_pending"
	}
	return ""
}

func (policy Policy) Admit(detection Detection, size int64) error {
	if size > int64(policy.MaximumFileMB)<<20 {
		return Failure("FILE_SIZE_LIMIT")
	}
	if _, known := Lookup(detection.Format); !known {
		return Failure("FILE_TYPE_UNKNOWN")
	}
	if detection.Mismatch && (policy.Unknown != "store_text" || detection.Format != "txt") {
		return Failure("FILE_TYPE_MISMATCH")
	}
	if !slices.Contains(policy.Store, detection.Format) {
		return Failure("FILE_FORMAT_DISABLED")
	}
	if policy.NonIndexable == "reject" && policy.IndexReason(detection, size) != "" {
		return Failure("FILE_NOT_INDEXABLE")
	}
	return nil
}
