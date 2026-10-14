package store

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestImportLifecycle(t *testing.T) {
	s, m, _ := newTestStore(t)
	ctx := context.Background()
	u := createUser(t, s, "a@example.com")
	other := createUser(t, s, "b@example.com")
	g, _ := s.CreateGroup(ctx, u.ID, "Group")

	if _, err := s.CreateImport(ctx, NewImport{UserID: other.ID, GroupID: g.ID, Filename: "x.cbz", Source: SourceUpload}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected other user's group to be rejected, got %v", err)
	}

	up, err := s.CreateImport(ctx, NewImport{UserID: u.ID, GroupID: g.ID, Filename: "Saga 01.cbz", Source: SourceUpload, UploadPath: "/spool/1/archive"})
	if err != nil {
		t.Fatal(err)
	}
	if up.Status != ImportQueued || up.Name != "Saga 01" || up.GroupName != "Group" || up.UploadPath != "/spool/1/archive" || up.CreatedAt.IsZero() {
		t.Fatalf("unexpected import: %+v", up)
	}
	ia, err := s.CreateImport(ctx, NewImport{UserID: u.ID, GroupID: g.ID, Filename: "Saga 02.cbr", Source: SourceArchive, ArchiveIdentifier: "saga", ArchiveName: "Saga 02.cbr"})
	if err != nil {
		t.Fatal(err)
	}

	paths, _ := s.PendingUploadPaths(ctx)
	if !paths["/spool/1/archive"] || len(paths) != 1 {
		t.Fatalf("pending upload paths: %v", paths)
	}

	// Jobs are claimed oldest first, once each.
	first, err := s.ClaimImport(ctx)
	if err != nil || first == nil || first.ID != up.ID || first.Status != ImportProcessing || first.UserID != u.ID {
		t.Fatalf("first claim: %+v %v", first, err)
	}
	second, _ := s.ClaimImport(ctx)
	if second == nil || second.ID != ia.ID || second.ArchiveIdentifier != "saga" {
		t.Fatalf("second claim: %+v", second)
	}
	if none, err := s.ClaimImport(ctx); none != nil || err != nil {
		t.Fatalf("expected nothing to claim, got %+v %v", none, err)
	}

	c := createComic(t, s, m, u.ID, g.ID, "Saga 01.cbz", 1)
	if err := s.FinishImport(ctx, up.ID, c.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.FailImport(ctx, ia.ID, "archive.org is down"); err != nil {
		t.Fatal(err)
	}
	// Finished jobs can't change state again.
	if err := s.RequeueImport(ctx, up.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected finished job not to requeue, got %v", err)
	}

	list, err := s.ListImports(ctx, u.ID)
	if err != nil || len(list) != 2 {
		t.Fatalf("list: %+v %v", list, err)
	}
	if list[0].ID != ia.ID || list[0].Status != ImportFailed || list[0].Error != "archive.org is down" {
		t.Fatalf("failed job: %+v", list[0])
	}
	if list[1].Status != ImportDone || list[1].ComicID == nil || *list[1].ComicID != c.ID || list[1].UploadPath != "" {
		t.Fatalf("done job: %+v", list[1])
	}
	if others, _ := s.ListImports(ctx, other.ID); len(others) != 0 {
		t.Fatalf("other user sees jobs: %+v", others)
	}

	// Deleting the comic keeps the job, without a comic.
	if err := s.DeleteComic(ctx, u.ID, c.ID); err != nil {
		t.Fatal(err)
	}
	if j, _ := s.GetImport(ctx, u.ID, up.ID); j.ComicID != nil {
		t.Fatalf("expected comic to be unlinked: %+v", j)
	}

	if _, err := s.DeleteImport(ctx, other.ID, ia.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected other user's delete to fail, got %v", err)
	}
	if j, err := s.DeleteImport(ctx, u.ID, ia.ID); err != nil || j.ID != ia.ID {
		t.Fatalf("delete: %+v %v", j, err)
	}

	// Pruning only removes old finished jobs.
	if err := s.PruneImports(ctx, time.Hour); err != nil {
		t.Fatal(err)
	}
	if list, _ := s.ListImports(ctx, u.ID); len(list) != 1 {
		t.Fatalf("recent job pruned: %+v", list)
	}
	s.db.Exec(`UPDATE imports SET updated_at = datetime('now', '-2 hours')`)
	s.PruneImports(ctx, time.Hour)
	if list, _ := s.ListImports(ctx, u.ID); len(list) != 0 {
		t.Fatalf("old job not pruned: %+v", list)
	}
}

func TestResetImports(t *testing.T) {
	s, _, _ := newTestStore(t)
	ctx := context.Background()
	u := createUser(t, s, "a@example.com")
	g, _ := s.CreateGroup(ctx, u.ID, "Group")
	j, _ := s.CreateImport(ctx, NewImport{UserID: u.ID, GroupID: g.ID, Filename: "x.cbz", Source: SourceArchive, ArchiveIdentifier: "x", ArchiveName: "x.cbz"})
	s.ClaimImport(ctx)

	// As after a crash mid-job.
	n, err := s.ResetImports(ctx)
	if err != nil || n != 1 {
		t.Fatalf("reset: %d %v", n, err)
	}
	if again, _ := s.ClaimImport(ctx); again == nil || again.ID != j.ID {
		t.Fatalf("expected job to be claimable again, got %+v", again)
	}

	// Deleting the group removes its jobs.
	if err := s.DeleteGroup(ctx, u.ID, g.ID); err != nil {
		t.Fatal(err)
	}
	if list, _ := s.ListImports(ctx, u.ID); len(list) != 0 {
		t.Fatalf("jobs left after group delete: %+v", list)
	}
}
