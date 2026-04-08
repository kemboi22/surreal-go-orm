package types

import "context"

type BeforeCreateHook interface {
	BeforeCreate(ctx context.Context, db any) error
}

type AfterCreateHook interface {
	AfterCreate(ctx context.Context, db any) error
}

type BeforeUpdateHook interface {
	BeforeUpdate(ctx context.Context, db any) error
}

type AfterUpdateHook interface {
	AfterUpdate(ctx context.Context, db any) error
}

type BeforeDeleteHook interface {
	BeforeDelete(ctx context.Context, db any) error
}

type AfterDeleteHook interface {
	AfterDelete(ctx context.Context, db any) error
}

type BeforeSaveHook interface {
	BeforeSave(ctx context.Context, db any) error
}

type AfterSaveHook interface {
	AfterSave(ctx context.Context, db any) error
}

func CallBeforeCreateHook[T any](ctx context.Context, db any, model T) error {
	if h, ok := any(model).(BeforeCreateHook); ok {
		return h.BeforeCreate(ctx, db)
	}
	return nil
}

func CallAfterCreateHook[T any](ctx context.Context, db any, model T) error {
	if h, ok := any(model).(AfterCreateHook); ok {
		return h.AfterCreate(ctx, db)
	}
	return nil
}

func CallBeforeUpdateHook[T any](ctx context.Context, db any, model T) error {
	if h, ok := any(model).(BeforeUpdateHook); ok {
		return h.BeforeUpdate(ctx, db)
	}
	return nil
}

func CallAfterUpdateHook[T any](ctx context.Context, db any, model T) error {
	if h, ok := any(model).(AfterUpdateHook); ok {
		return h.AfterUpdate(ctx, db)
	}
	return nil
}

func CallBeforeDeleteHook[T any](ctx context.Context, db any, model T) error {
	if h, ok := any(model).(BeforeDeleteHook); ok {
		return h.BeforeDelete(ctx, db)
	}
	return nil
}

func CallAfterDeleteHook[T any](ctx context.Context, db any, model T) error {
	if h, ok := any(model).(AfterDeleteHook); ok {
		return h.AfterDelete(ctx, db)
	}
	return nil
}

func CallBeforeSaveHook[T any](ctx context.Context, db any, model T) error {
	if h, ok := any(model).(BeforeSaveHook); ok {
		return h.BeforeSave(ctx, db)
	}
	return nil
}

func CallAfterSaveHook[T any](ctx context.Context, db any, model T) error {
	if h, ok := any(model).(AfterSaveHook); ok {
		return h.AfterSave(ctx, db)
	}
	return nil
}
