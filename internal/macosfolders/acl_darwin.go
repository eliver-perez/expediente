//go:build darwin && cgo

package macosfolders

/*
#include <sys/acl.h>
#include <membership.h>
#include <uuid/uuid.h>
#include <errno.h>
#include <string.h>

// Cada fallo identifica la instrucción exacta, sin seguir con una ACL parcial.
#define AIDEX_ACL_TRY(call) do { *operation = #call; if ((call) != 0) { result = errno; acl_free(acl); return result; } } while (0)

// Se añade únicamente un allow para el UUID indicado. No se eliminan ACL,
// denies, propietarios ni permisos POSIX de otras personas.
static int aidex_allow_fd(int fd, unsigned int uid, int mode, int flags, const char **operation) {
    *operation = "mbr_uid_to_uuid";
    uuid_t uuid;
    int result = mbr_uid_to_uuid(uid, uuid);
    if (result != 0) return result;
    *operation = "acl_get_fd_np";
    acl_t acl = acl_get_fd_np(fd, ACL_TYPE_EXTENDED);
    if (!acl) {
        if (errno != ENOENT) return errno;
        *operation = "acl_init";
        acl = acl_init(2);
        if (!acl) return errno;
    }
    acl_entry_t entry = NULL, found = NULL;
    int index = ACL_FIRST_ENTRY;
    while (acl_get_entry(acl, index, &entry) == 0) {
        index = ACL_NEXT_ENTRY;
        acl_tag_t tag;
        acl_flagset_t set;
        if (acl_get_tag_type(entry, &tag) != 0 || tag != ACL_EXTENDED_ALLOW) continue;
        void *qualifier = acl_get_qualifier(entry);
        int same = qualifier && memcmp(qualifier, uuid, sizeof(uuid)) == 0;
        if (qualifier) acl_free(qualifier);
        if (!same || acl_get_flagset_np(entry, &set) != 0) continue;
        int mask = 0;
        if (acl_get_flag_np(set, ACL_ENTRY_FILE_INHERIT)) mask |= 1;
        if (acl_get_flag_np(set, ACL_ENTRY_DIRECTORY_INHERIT)) mask |= 2;
        if (acl_get_flag_np(set, ACL_ENTRY_ONLY_INHERIT)) mask |= 4;
        if (acl_get_flag_np(set, ACL_ENTRY_INHERITED)) continue;
        if (mask == flags) { found = entry; break; }
    }
    if (!found) {
        AIDEX_ACL_TRY(acl_create_entry(&acl, &found));
        AIDEX_ACL_TRY(acl_set_tag_type(found, ACL_EXTENDED_ALLOW));
        AIDEX_ACL_TRY(acl_set_qualifier(found, uuid));
    }
    acl_permset_t permissions;
    acl_flagset_t inherited;
    AIDEX_ACL_TRY(acl_get_permset(found, &permissions));
    AIDEX_ACL_TRY(acl_get_flagset_np(found, &inherited));
    // mode: 0 recorrido; 1 archivo lector; 2 directorio lector;
    // 3 archivo administrado; 4 directorio administrado.
    if (mode == 0) AIDEX_ACL_TRY(acl_add_perm(permissions, ACL_SEARCH));
    else {
        AIDEX_ACL_TRY(acl_add_perm(permissions, ACL_READ_DATA));
        AIDEX_ACL_TRY(acl_add_perm(permissions, ACL_READ_ATTRIBUTES));
        AIDEX_ACL_TRY(acl_add_perm(permissions, ACL_READ_EXTATTRIBUTES));
        AIDEX_ACL_TRY(acl_add_perm(permissions, ACL_READ_SECURITY));
        if (mode == 2 || mode == 4) AIDEX_ACL_TRY(acl_add_perm(permissions, ACL_SEARCH));
        if (mode == 3 || mode == 4) {
            AIDEX_ACL_TRY(acl_add_perm(permissions, ACL_WRITE_DATA));
            AIDEX_ACL_TRY(acl_add_perm(permissions, ACL_APPEND_DATA));
            AIDEX_ACL_TRY(acl_add_perm(permissions, ACL_WRITE_ATTRIBUTES));
            AIDEX_ACL_TRY(acl_add_perm(permissions, ACL_WRITE_EXTATTRIBUTES));
            AIDEX_ACL_TRY(acl_add_perm(permissions, mode == 4 ? ACL_DELETE_CHILD : ACL_DELETE));
        }
    }
    if (flags & 1) AIDEX_ACL_TRY(acl_add_flag_np(inherited, ACL_ENTRY_FILE_INHERIT));
    if (flags & 2) AIDEX_ACL_TRY(acl_add_flag_np(inherited, ACL_ENTRY_DIRECTORY_INHERIT));
    if (flags & 4) AIDEX_ACL_TRY(acl_add_flag_np(inherited, ACL_ENTRY_ONLY_INHERIT));
    *operation = "acl_set_fd_np";
    result=acl_set_fd_np(fd, acl, ACL_TYPE_EXTENDED)==0 ? 0 : errno;
    acl_free(acl);
    return result;
}
#undef AIDEX_ACL_TRY
*/
import "C"

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"syscall"
)

type permissionEntry struct {
	path string
	info fs.FileInfo
}

// Authorize exige root; el llamador valida además el ticket y confirma la ruta
// mediante la aplicación nativa. No acepta comandos ni usuarios arbitrarios.
func Authorize(ctx context.Context, selection Selection, serviceUID, finderUID int, probe TraversalProbe) (int, error) {
	if os.Geteuid() != 0 {
		return 0, fmt.Errorf("se requiere autorización administrativa de macOS")
	}
	if err := ValidateSelection(selection); err != nil {
		return 0, err
	}
	return authorizeTree(ctx, selection, serviceUID, finderUID, probe)
}

func authorizeTree(ctx context.Context, selection Selection, serviceUID, finderUID int, probe TraversalProbe) (int, error) {
	return authorizeTreeMode(ctx, selection, serviceUID, finderUID, false, probe)
}

// ExposePreservedDocuments solo admite la ubicación fija del desinstalador.
// Los enlaces archivados permanecen intactos y jamás se siguen ni autorizan.
func ExposePreservedDocuments(ctx context.Context, finderUID int, probe TraversalProbe) (int, error) {
	if os.Geteuid() != 0 {
		return 0, fmt.Errorf("se requiere autorización administrativa")
	}
	path := "/Library/Application Support/AIDEX/Documentos conservados"
	if _, err := os.Lstat(path); os.IsNotExist(err) {
		return 0, nil
	}
	return authorizeTreeMode(ctx, Selection{Path: path, Source: "linked"}, finderUID, finderUID, true, probe)
}

func authorizeTreeMode(ctx context.Context, selection Selection, serviceUID, finderUID int, archive bool, probe TraversalProbe) (int, error) {
	var ancestors []permissionEntry
	var selectedInfo fs.FileInfo
	for current := selection.Path; current != "/"; current = filepath.Dir(current) {
		info, err := os.Lstat(current)
		if err != nil {
			return 0, operationFailure("lstat_selection_ancestor", current, err)
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return 0, fmt.Errorf("enlace o directorio no válido en la ruta")
		}
		if current != selection.Path {
			ancestors = append(ancestors, permissionEntry{current, info})
		} else {
			selectedInfo = info
		}
	}
	root, err := os.OpenRoot(selection.Path)
	if err != nil {
		return 0, operationFailure("open_selected_root", selection.Path, err)
	}
	defer root.Close()
	info, err := root.Stat(".")
	if err != nil {
		return 0, operationFailure("fstat_selected_root", selection.Path, err)
	}
	if !os.SameFile(selectedInfo, info) {
		return 0, fmt.Errorf("la carpeta seleccionada cambió antes de abrirla; no se modificaron permisos")
	}
	device := info.Sys().(*syscall.Stat_t).Dev
	entries := []permissionEntry{}
	err = fs.WalkDir(root.FS(), ".", func(path string, entry fs.DirEntry, walkErr error) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if walkErr != nil {
			return operationFailure("inventory", filepath.Join(selection.Path, path), walkErr)
		}
		info, err := entry.Info()
		if err != nil {
			return operationFailure("inventory_stat", filepath.Join(selection.Path, path), err)
		}
		stat := info.Sys().(*syscall.Stat_t)
		if archive && (info.Mode()&os.ModeSymlink != 0 || !info.IsDir() && stat.Nlink > 1) {
			return nil
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() && !info.Mode().IsRegular() || stat.Dev != device || !info.IsDir() && stat.Nlink > 1 {
			return fmt.Errorf("la carpeta contiene enlaces, archivos especiales o montajes internos; no se modificaron sus permisos")
		}
		entries = append(entries, permissionEntry{path, info})
		if len(entries) > 200000 {
			return fmt.Errorf("la autorización supera 200000 entradas; selecciona una carpeta más concreta")
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	// Comprobar desde fuera hacia dentro con la identidad real del destinatario.
	// No escribir en /Users, /Volumes ni otros superiores que ya permiten llegar
	// a la carpeta: además de innecesario, puede fallar en ubicaciones protegidas.
	if probe == nil {
		return 0, operationFailure("traversal_probe_missing", selection.Path, fmt.Errorf("falta la comprobación de recorrido"))
	}
	for index := len(ancestors) - 1; index >= 0; index-- {
		ancestor := ancestors[index]
		if err = ctx.Err(); err != nil {
			return 0, err
		}

		identities := []int{serviceUID}
		if selection.Source == "managed" && finderUID != serviceUID {
			identities = append(identities, finderUID)
		}
		for _, identity := range identities {
			var accessible bool
			accessible, err = probe(ctx, identity, ancestor.path)
			if err != nil {
				err = operationFailure("check_ancestor_traverse", ancestor.path, err)
				break
			}
			if accessible {
				slog.Info("folder authorization", "operation", "ancestor_already_traversable", "path", ancestor.path, "target_uid", identity, "effective_uid", os.Geteuid())
				continue
			}
			// Abrir para modificar únicamente después de confirmar que falta recorrido.
			file, openErr := os.Open(ancestor.path)
			if openErr != nil {
				err = operationFailure("open_ancestor", ancestor.path, openErr)
				break
			}
			after, statErr := file.Stat()
			if statErr != nil {
				file.Close()
				err = operationFailure("fstat_ancestor", ancestor.path, statErr)
				break
			}
			if !os.SameFile(ancestor.info, after) {
				file.Close()
				err = operationFailure("validate_ancestor", ancestor.path, fmt.Errorf("cambió un directorio superior"))
				break
			}
			err = allow(file, identity, 0, 0)
			file.Close()
			if err != nil {
				break
			}
			accessible, err = probe(ctx, identity, ancestor.path)
			if err == nil && !accessible {
				err = syscall.EACCES
			}
			if err != nil {
				err = operationFailure("verify_ancestor_traverse", ancestor.path, err)
				break
			}
		}
		if err != nil {
			return 0, err
		}
	}

	completed := 0
	for _, entry := range entries {
		if err = ctx.Err(); err != nil {
			return completed, err
		}
		before, lstatErr := root.Lstat(entry.path)
		if lstatErr != nil {
			return completed, operationFailure("lstat_selected_entry", filepath.Join(selection.Path, entry.path), lstatErr)
		}
		if !os.SameFile(before, entry.info) || before.Mode()&os.ModeSymlink != 0 {
			return completed, operationFailure("validate_selected_entry", filepath.Join(selection.Path, entry.path), fmt.Errorf("cambió una entrada durante la autorización"))
		}
		file, openErr := root.Open(entry.path)
		if openErr != nil {
			return completed, operationFailure("open_selected_entry", filepath.Join(selection.Path, entry.path), openErr)
		}
		after, statErr := file.Stat()
		if statErr != nil {
			file.Close()
			return completed, operationFailure("fstat_selected_entry", filepath.Join(selection.Path, entry.path), statErr)
		}
		if !os.SameFile(entry.info, after) || !after.IsDir() && after.Sys().(*syscall.Stat_t).Nlink > 1 {
			file.Close()
			return completed, fmt.Errorf("cambió una identidad de archivo")
		}
		fileMode, dirMode := 1, 2
		if selection.Source == "managed" {
			fileMode, dirMode = 3, 4
		}
		if entry.info.IsDir() {
			err = allow(file, serviceUID, dirMode, 2)
			if err == nil {
				err = allow(file, serviceUID, fileMode, 7)
			}
			if err == nil && selection.Source == "managed" {
				err = allow(file, finderUID, 2, 2)
				if err == nil {
					err = allow(file, finderUID, 1, 7)
				}
			}
		} else {
			err = allow(file, serviceUID, fileMode, 0)
			if err == nil && selection.Source == "managed" {
				err = allow(file, finderUID, 1, 0)
			}
		}
		file.Close()
		if err != nil {
			return completed, err
		}
		completed++
	}
	return completed, nil
}

func allow(file *os.File, uid, mode, flags int) error {
	var operation *C.char
	if code := C.aidex_allow_fd(C.int(file.Fd()), C.uint(uid), C.int(mode), C.int(flags), &operation); code != 0 {
		slog.Error("folder permission grant failed", "path", file.Name(), "target_uid", uid, "permission_mode", mode, "inheritance_flags", flags, "operation", C.GoString(operation), "errno", int(code), "effective_uid", os.Geteuid())
		return operationFailure(C.GoString(operation), file.Name(), syscall.Errno(code))
	}
	return nil
}
