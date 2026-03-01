package smiwrite

import (
	"fmt"

	"github.com/golangsnmp/gomib/mib"
)

// emitNotificationType writes a NOTIFICATION-TYPE definition.
// All notifications (including those originating from TRAP-TYPE) are emitted
// as NOTIFICATION-TYPE.
func (e *emitter) emitNotificationType(notif *mib.Notification) error {
	if _, err := fmt.Fprintf(e.w, "%s NOTIFICATION-TYPE\n", notif.Name()); err != nil {
		return err
	}

	// OBJECTS clause
	if objects := notif.Objects(); len(objects) > 0 {
		names := make([]string, len(objects))
		for i, obj := range objects {
			names[i] = obj.Name()
		}
		if err := e.writeInlineList("    ", "OBJECTS", names); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintf(e.w, "    STATUS %s\n", formatStatus(notif.Status())); err != nil {
		return err
	}

	if err := e.writeDescription(notif.Description()); err != nil {
		return err
	}

	if ref := notif.Reference(); ref != "" {
		if _, err := fmt.Fprintf(e.w, "    REFERENCE\n        %s\n", formatDescription(ref)); err != nil {
			return err
		}
	}

	nd := notif.Node()
	_, err := fmt.Fprintf(e.w, "    ::= %s\n", formatOIDAssignment(nd))
	return err
}
