package surrealgoorm

const (
	hookSaving   = "saving"
	hookSaved    = "saved"
	hookCreating = "creating"
	hookCreated  = "created"
	hookUpdating = "updating"
	hookUpdated  = "updated"
	hookDeleting = "deleting"
	hookDeleted  = "deleted"
)

type SavingHook interface {
	Saving() error
}
type SavedHook interface {
	Saved()
}
type CreatingHook interface {
	Creating() error
}
type CreatedHook interface {
	Created()
}
type UpdatingHook interface {
	Updating() error
}
type UpdatedHook interface {
	Updated()
}
type DeletingHook interface {
	Deleting() error
}
type DeletedHook interface {
	Deleted()
}

func fireHook(v any, event string) error {
	switch event {
	case hookSaving:
		if h, ok := v.(SavingHook); ok {
			return h.Saving()
		}
	case hookCreating:
		if h, ok := v.(CreatingHook); ok {
			return h.Creating()
		}
	case hookUpdating:
		if h, ok := v.(UpdatingHook); ok {
			return h.Updating()
		}
	case hookDeleting:
		if h, ok := v.(DeletingHook); ok {
			return h.Deleting()
		}
	case hookCreated:
		if h, ok := v.(CreatedHook); ok {
			h.Created()
		}
	case hookUpdated:
		if h, ok := v.(UpdatedHook); ok {
			h.Updated()
		}
	case hookDeleted:
		if h, ok := v.(DeletedHook); ok {
			h.Deleted()
		}
	case hookSaved:
		if h, ok := v.(SavedHook); ok {
			h.Saved()
		}
	}
	return nil
}
