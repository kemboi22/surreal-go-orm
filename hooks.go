package surrealgoorm

import (
	"context"

	ormtypes "github.com/kemboi22/surreal-go-orm/types"
)

func callBeforeCreate(ctx context.Context, db *DB, model any) error {
	if err := ormtypes.CallBeforeSaveHook(ctx, db, model); err != nil {
		return err
	}
	return ormtypes.CallBeforeCreateHook(ctx, db, model)
}

func callAfterCreate(ctx context.Context, db *DB, model any) error {
	if err := ormtypes.CallAfterCreateHook(ctx, db, model); err != nil {
		return err
	}
	return ormtypes.CallAfterSaveHook(ctx, db, model)
}

func callBeforeUpdate(ctx context.Context, db *DB, model any) error {
	if err := ormtypes.CallBeforeSaveHook(ctx, db, model); err != nil {
		return err
	}
	return ormtypes.CallBeforeUpdateHook(ctx, db, model)
}

func callAfterUpdate(ctx context.Context, db *DB, model any) error {
	if err := ormtypes.CallAfterUpdateHook(ctx, db, model); err != nil {
		return err
	}
	return ormtypes.CallAfterSaveHook(ctx, db, model)
}

func callBeforeDelete(ctx context.Context, db *DB, model any) error {
	return ormtypes.CallBeforeDeleteHook(ctx, db, model)
}

func callAfterDelete(ctx context.Context, db *DB, model any) error {
	return ormtypes.CallAfterDeleteHook(ctx, db, model)
}
