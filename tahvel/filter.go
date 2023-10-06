package tahvel

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"
	"unicode"

	bb "github.com/jtagcat/util/bbolt"
	"github.com/jtagcat/util/std"
	"go.etcd.io/bbolt"
)

var ADMIN_MAIL = os.Getenv("ADMIN_MAIL")

func (u *User) UnknownACLs() (groups []string) {
	for _, role := range u.Roles {
		if role.ignorable() {
			continue
		}

		if len(role.ACLs()) == 0 {
			slog.Warn("Unknown ACL group", slog.String("studentGroup", role.StudentGroup), slog.String("user", u.FullName))
			groups = append(groups, fmt.Sprintf("Kasutaja roll %s on kaardistamata õigustega. Palun kirjuta %s, et filtreerida enda õigustega ruume.", role.StudentGroup, ADMIN_MAIL))
		}
	}

	return
}

// The role gives access to nothing.
//
//	Adding it to aclMapping would mess with
//	'has no identifyiable roles' check.
func (r *UserRole) ignorable() bool {
	return r.Id == 223113 // booking role
}

func (u *User) ACLCompositeName() string {
	return aclCompositeName(u.Roles)
}

func aclCompositeName(roles []UserRole) string {
	var composite []string
	for _, role := range roles {
		if role.ignorable() {
			continue
		}

		composite = append(composite, role.StudentGroup)
	}

	return strings.Join(composite, ",")
}

func aclPrefix(full string) (p string) {
	p, _, _ = std.CutFunc(full, unicode.IsDigit)
	p = strings.TrimSuffix(p, "-")

	p = strings.TrimPrefix(p, "eõ-") // TODO: kas on vahet?
	p = strings.TrimPrefix(p, "JÕ-") // TODO: kas on vahet?

	return
}

func (r *UserRole) ACLs() []string {
	if r.ignorable() {
		return nil
	}

	acls, ok := aclMapping[aclPrefix(r.StudentGroup)]
	if !ok {
		return nil
	}

	return acls
}

func (r *Room) hasAccess(db *bbolt.DB, roles []UserRole) bool {
	if crowd := bb.Get(db, []byte("crowdsourced_room_acl"), fmt.Sprintf("%d:%s", r.Id, aclCompositeName(roles))); crowd != "" {
		if crowd == "1" {
			return true
		}
		return false
	}

	var personACLs []string
	for _, role := range roles {
		personACLs = append(personACLs, role.ACLs()...)
	}

	if len(personACLs) == 0 {
		return true
	}

	roomAcl, ok := roomAclMapping[r.OnlyCode()]
	if !ok {
		slog.Debug("room is missing ACL", slog.String("onlyCode", r.OnlyCode()), slog.Int("id", r.Id), slog.String("fullName", r.RoomCode+": "+r.RoomName))
		r.MissingACL = true
		return true
	}

	for _, personACL := range personACLs {
		if personACL == roomAcl {
			return true
		}
	}

	return false
}

// // if !r.needsOk(needsOneOf) {
// // 	continue
// // }
// func (r *Room) hasOneOf(candidates []string) bool {
// 	if len(oneOf) == 0 {
// 		return true
// 	}

// 	for _, roomHas := range r.FlatEquipment() {
// 		for _, need := range oneOf {
// 			if roomHas == need {
// 				return true
// 			}
// 		}
// 	}

// 	return false
// }

func (r *Room) FlatEquipment() (flat []string) {
	for _, e := range r.Equipment {
		flat = append(flat, e.Equipment)
	}

	return
}

const DICKTRESHOLD = 5 * time.Hour

func FilterRooms(db *bbolt.DB, rooms []Room,
	aclGroups []UserRole, needsPiano bool,
	bookStart, bookStop string, fuzzy time.Duration,
) (good []Room, conflicting []Room, dicks []string) {
	startT, _ := time.Parse("15:04", bookStart)
	stopT, _ := time.Parse("15:04", bookStop)

	startT.Add(fuzzy)
	stopT.Add(-fuzzy)
	stopIsSet := stopT.Unix() != -62167219200

room:
	for _, r := range rooms {
		if !r.IsUsedInStudy {
			continue
		}

		if !r.hasAccess(db, aclGroups) {
			continue
		}

		if needsPiano && r.PianoCount < 1 {
			continue
		}

		for _, booking := range r.Times {
			bStart, bStop, ok := strings.Cut(booking, " - ")
			if !ok {
				slog.Warn("found unusual booking time", slog.String("anomaly", booking))
				continue
			}

			bStartT, err := time.Parse("15:04", bStart)
			if err != nil {
				slog.Warn("found unusual booking time start", slog.String("anomaly", booking))
				continue
			}
			bStopT, err := time.Parse("15:04", bStop)
			if err != nil {
				slog.Warn("found unusual booking time stop", slog.String("anomaly", booking))
				continue
			}

			bDuration := bStopT.Sub(bStartT)
			if bDuration >= DICKTRESHOLD {
				dicks = append(dicks, fmt.Sprintf("%s on %s broneeritud %s", r.RoomCode, bDuration, booking))
			}

			// filtering out non-conflicting times
			if bStopT.Before(startT) {
				continue
			}
			if stopIsSet && bStartT.After(stopT) {
				continue
			}

			r.ConflictReason = fmt.Sprintf("%s on kinni %s (%s)", r.RoomCode, booking, bDuration)
			conflicting = append(conflicting, r)
			continue room
		}

		good = append(good, r)
	}

	return
}

func FilterEquipmentReferenced(equipment map[string]string, rooms []Room) map[string]string {
	referenced := make(map[string]string)

	for _, r := range rooms {
		for _, e := range r.FlatEquipment() {
			referenced[e] = equipment[e]
		}
	}

	return referenced
}
