package repository

import (
	"context"
	"errors"
	"time"

	"backend/internal/domain"
	"backend/internal/repository/database"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type decisionRepository struct {
	db *pgxpool.Pool
	q  *database.Queries
}

func NewDecisionRepository(db *pgxpool.Pool) domain.DecisionRepository {
	return &decisionRepository{
		db: db,
		q:  database.New(db),
	}
}

func (r *decisionRepository) Create(ctx context.Context, d domain.Decision, initialOptions []domain.Option, initialTagIDs []uuid.UUID) (domain.Decision, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return domain.Decision{}, err
	}
	defer tx.Rollback(ctx)

	qtx := r.q.WithTx(tx)

	var contextText pgtype.Text
	if d.Context != nil {
		contextText = pgtype.Text{String: *d.Context, Valid: true}
	}
	var conf pgtype.Int4
	if d.ConfidenceBefore != nil {
		conf = pgtype.Int4{Int32: int32(*d.ConfidenceBefore), Valid: true}
	}
	var reviewDate pgtype.Date
	if d.ScheduledReviewDate != nil {
		reviewDate = pgtype.Date{Time: *d.ScheduledReviewDate, Valid: true}
	}

	var createdAtTS pgtype.Timestamptz
	if !d.CreatedAt.IsZero() {
		createdAtTS = pgtype.Timestamptz{Time: d.CreatedAt, Valid: true}
	}

	var idempText pgtype.Text
	if d.IdempotencyKey != nil && *d.IdempotencyKey != "" {
		idempText = pgtype.Text{String: *d.IdempotencyKey, Valid: true}
	}

	createdRow, err := qtx.CreateDecision(ctx, database.CreateDecisionParams{
		UserID:              d.UserID,
		Title:               d.Title,
		Context:             contextText,
		ConfidenceBefore:    conf,
		ScheduledReviewDate: reviewDate,
		CreatedAt:           createdAtTS,
		IdempotencyKey:      idempText,
	})
	if err != nil {
		return domain.Decision{}, err
	}

	createdOptions := make([]domain.Option, 0, len(initialOptions))
	for _, opt := range initialOptions {
		var prosText pgtype.Text
		if opt.Pros != nil {
			prosText = pgtype.Text{String: *opt.Pros, Valid: true}
		}
		var consText pgtype.Text
		if opt.Cons != nil {
			consText = pgtype.Text{String: *opt.Cons, Valid: true}
		}

		optRow, err := qtx.CreateOption(ctx, database.CreateOptionParams{
			DecisionID: createdRow.ID,
			Title:      opt.Title,
			Pros:       prosText,
			Cons:       consText,
		})
		if err != nil {
			return domain.Decision{}, err
		}

		var pStr, cStr *string
		if optRow.Pros.Valid {
			pStr = &optRow.Pros.String
		}
		if optRow.Cons.Valid {
			cStr = &optRow.Cons.String
		}

		createdOptions = append(createdOptions, domain.Option{
			ID:         optRow.ID,
			DecisionID: optRow.DecisionID,
			Title:      optRow.Title,
			Pros:       pStr,
			Cons:       cStr,
			CreatedAt:  optRow.CreatedAt.Time,
		})
	}

	createdTags := make([]domain.Tag, 0, len(initialTagIDs))
	for _, tagID := range initialTagIDs {
		_, err := qtx.AddTagToDecision(ctx, database.AddTagToDecisionParams{
			DecisionID: createdRow.ID,
			TagID:      tagID,
			UserID:     d.UserID,
		})
		if err != nil {
			return domain.Decision{}, err
		}
	}

	if len(initialTagIDs) > 0 {
		tagRows, err := qtx.GetTagsForDecision(ctx, createdRow.ID)
		if err != nil {
			return domain.Decision{}, err
		}
		for _, t := range tagRows {
			createdTags = append(createdTags, domain.Tag{
				ID:       t.ID,
				UserID:   d.UserID,
				Name:     t.Name,
				ColorHex: t.ColorHex.String,
			})
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.Decision{}, err
	}

	var ctxStr *string
	if createdRow.Context.Valid {
		ctxStr = &createdRow.Context.String
	}
	var cInt *int
	if createdRow.ConfidenceBefore.Valid {
		v := int(createdRow.ConfidenceBefore.Int32)
		cInt = &v
	}
	var rDate *time.Time
	if createdRow.ScheduledReviewDate.Valid {
		t := createdRow.ScheduledReviewDate.Time
		rDate = &t
	}

	return domain.Decision{
		ID:                  createdRow.ID,
		UserID:              createdRow.UserID,
		Title:               createdRow.Title,
		Context:             ctxStr,
		ConfidenceBefore:    cInt,
		ScheduledReviewDate: rDate,
		CreatedAt:           createdRow.CreatedAt.Time,
		UpdatedAt:           createdRow.UpdatedAt.Time,
		ServerUpdatedAt:     createdRow.ServerUpdatedAt.Time,
		Options:             createdOptions,
		Tags:                createdTags,
	}, nil
}

func (r *decisionRepository) GetByID(ctx context.Context, id, userID uuid.UUID) (domain.Decision, error) {
	row, err := r.q.GetDecisionByID(ctx, database.GetDecisionByIDParams{
		ID:     id,
		UserID: userID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Decision{}, domain.ErrDecisionNotFound
		}
		return domain.Decision{}, err
	}

	if row.DeletedAt.Valid {
		return domain.Decision{}, domain.ErrDecisionNotFound
	}

	return r.populateRelations(ctx, row)
}

func (r *decisionRepository) GetByIdempotencyKey(ctx context.Context, userID uuid.UUID, key string) (domain.Decision, error) {
	row, err := r.q.GetDecisionByIdempotencyKey(ctx, database.GetDecisionByIdempotencyKeyParams{
		UserID:         userID,
		IdempotencyKey: pgtype.Text{String: key, Valid: true},
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Decision{}, domain.ErrDecisionNotFound
		}
		return domain.Decision{}, err
	}

	return r.populateRelations(ctx, row)
}

func (r *decisionRepository) GetActiveByUserID(ctx context.Context, userID uuid.UUID) ([]domain.Decision, error) {
	rows, err := r.q.GetActiveDecisionsByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	decisions := make([]domain.Decision, 0, len(rows))
	for _, row := range rows {
		d, err := r.populateRelations(ctx, row)
		if err != nil {
			return nil, err
		}
		decisions = append(decisions, d)
	}
	return decisions, nil
}

func (r *decisionRepository) ListUpdatedSince(ctx context.Context, userID uuid.UUID, updatedSince time.Time) ([]domain.Decision, error) {
	rows, err := r.q.ListDecisionsUpdatedSince(ctx, database.ListDecisionsUpdatedSinceParams{
		UserID:          userID,
		ServerUpdatedAt: pgtype.Timestamptz{Time: updatedSince, Valid: true},
	})
	if err != nil {
		return nil, err
	}

	decisions := make([]domain.Decision, 0, len(rows))
	for _, row := range rows {
		d, err := r.populateRelations(ctx, row)
		if err != nil {
			return nil, err
		}
		decisions = append(decisions, d)
	}
	return decisions, nil
}

func (r *decisionRepository) UpdateWithLWW(ctx context.Context, d domain.Decision, localUpdatedAt time.Time) (domain.Decision, error) {
	var contextText pgtype.Text
	if d.Context != nil {
		contextText = pgtype.Text{String: *d.Context, Valid: true}
	}
	var conf pgtype.Int4
	if d.ConfidenceBefore != nil {
		conf = pgtype.Int4{Int32: int32(*d.ConfidenceBefore), Valid: true}
	}
	var reviewDate pgtype.Date
	if d.ScheduledReviewDate != nil {
		reviewDate = pgtype.Date{Time: *d.ScheduledReviewDate, Valid: true}
	}

	updatedRow, err := r.q.UpdateDecisionWithLWW(ctx, database.UpdateDecisionWithLWWParams{
		ID:                  d.ID,
		UserID:              d.UserID,
		Title:               d.Title,
		Context:             contextText,
		ConfidenceBefore:    conf,
		ScheduledReviewDate: reviewDate,
		UpdatedAt:           pgtype.Timestamptz{Time: localUpdatedAt, Valid: true},
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			rawRow, getErr := r.q.GetDecisionByID(ctx, database.GetDecisionByIDParams{ID: d.ID, UserID: d.UserID})
			if getErr != nil {
				if errors.Is(getErr, pgx.ErrNoRows) {
					return domain.Decision{}, domain.ErrDecisionNotFound
				}
				return domain.Decision{}, getErr
			}
			serverDecision, popErr := r.populateRelations(ctx, rawRow)
			if popErr != nil {
				return domain.Decision{}, popErr
			}
			return domain.Decision{}, &domain.ErrConflict{ServerDecision: serverDecision}
		}
		return domain.Decision{}, err
	}

	return r.populateRelations(ctx, updatedRow)
}

func (r *decisionRepository) SoftDeleteWithLWW(ctx context.Context, id, userID uuid.UUID, localUpdatedAt *time.Time) error {
	var localTS pgtype.Timestamptz
	if localUpdatedAt != nil {
		localTS = pgtype.Timestamptz{Time: *localUpdatedAt, Valid: true}
	}

	rowsAffected, err := r.q.SoftDeleteDecisionWithLWW(ctx, database.SoftDeleteDecisionWithLWWParams{
		ID:             id,
		UserID:         userID,
		LocalUpdatedAt: localTS,
	})
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		rawRow, getErr := r.q.GetDecisionByID(ctx, database.GetDecisionByIDParams{ID: id, UserID: userID})
		if getErr != nil {
			if errors.Is(getErr, pgx.ErrNoRows) {
				return domain.ErrDecisionNotFound
			}
			return getErr
		}
		// Idempotent delete: if already soft-deleted, return success
		if rawRow.DeletedAt.Valid {
			return nil
		}
		serverDecision, popErr := r.populateRelations(ctx, rawRow)
		if popErr != nil {
			return popErr
		}
		return &domain.ErrConflict{ServerDecision: serverDecision}
	}

	return nil
}

func (r *decisionRepository) CreateOption(ctx context.Context, opt domain.Option, userID uuid.UUID) (domain.Option, error) {
	var prosText pgtype.Text
	if opt.Pros != nil {
		prosText = pgtype.Text{String: *opt.Pros, Valid: true}
	}
	var consText pgtype.Text
	if opt.Cons != nil {
		consText = pgtype.Text{String: *opt.Cons, Valid: true}
	}

	var row database.Option
	err := r.inTx(ctx, func(q *database.Queries) error {
		var err error
		row, err = q.CreateOption(ctx, database.CreateOptionParams{
			DecisionID: opt.DecisionID,
			Title:      opt.Title,
			Pros:       prosText,
			Cons:       consText,
		})
		if err != nil {
			return err
		}
		return q.TouchDecision(ctx, database.TouchDecisionParams{ID: opt.DecisionID, UserID: userID})
	})
	if err != nil {
		return domain.Option{}, err
	}

	var pStr, cStr *string
	if row.Pros.Valid {
		pStr = &row.Pros.String
	}
	if row.Cons.Valid {
		cStr = &row.Cons.String
	}

	return domain.Option{
		ID:         row.ID,
		DecisionID: row.DecisionID,
		Title:      row.Title,
		Pros:       pStr,
		Cons:       cStr,
		CreatedAt:  row.CreatedAt.Time,
	}, nil
}

func (r *decisionRepository) DeleteOption(ctx context.Context, optionID, decisionID, userID uuid.UUID) error {
	// Verify decision belongs to user
	if _, err := r.GetByID(ctx, decisionID, userID); err != nil {
		return err
	}

	// Check if this option is chosen in outcome
	outcome, err := r.q.GetDecisionOutcome(ctx, decisionID)
	if err == nil && outcome.ChosenOptionID == optionID {
		return domain.ErrCannotDeleteChosenOption
	}

	return r.inTx(ctx, func(q *database.Queries) error {
		if err := q.DeleteOptionByID(ctx, database.DeleteOptionByIDParams{
			ID:         optionID,
			DecisionID: decisionID,
		}); err != nil {
			return err
		}
		return q.TouchDecision(ctx, database.TouchDecisionParams{ID: decisionID, UserID: userID})
	})
}

func (r *decisionRepository) GetOptionByID(ctx context.Context, optionID, decisionID uuid.UUID) (domain.Option, error) {
	row, err := r.q.GetOptionByID(ctx, database.GetOptionByIDParams{
		ID:         optionID,
		DecisionID: decisionID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Option{}, errors.New("option not found")
		}
		return domain.Option{}, err
	}

	var pStr, cStr *string
	if row.Pros.Valid {
		pStr = &row.Pros.String
	}
	if row.Cons.Valid {
		cStr = &row.Cons.String
	}

	return domain.Option{
		ID:         row.ID,
		DecisionID: row.DecisionID,
		Title:      row.Title,
		Pros:       pStr,
		Cons:       cStr,
		CreatedAt:  row.CreatedAt.Time,
	}, nil
}

func (r *decisionRepository) UpsertOutcome(ctx context.Context, outcome domain.DecisionOutcome, userID uuid.UUID) (domain.DecisionOutcome, error) {
	if _, err := r.GetByID(ctx, outcome.DecisionID, userID); err != nil {
		return domain.DecisionOutcome{}, err
	}

	if _, err := r.GetOptionByID(ctx, outcome.ChosenOptionID, outcome.DecisionID); err != nil {
		return domain.DecisionOutcome{}, errors.New("chosen option does not belong to this decision")
	}

	var note pgtype.Text
	if outcome.CommitmentNote != nil {
		note = pgtype.Text{String: *outcome.CommitmentNote, Valid: true}
	}

	var decidedAtTS pgtype.Timestamptz
	if !outcome.DecidedAt.IsZero() {
		decidedAtTS = pgtype.Timestamptz{Time: outcome.DecidedAt, Valid: true}
	}

	var row database.DecisionOutcome
	err := r.inTx(ctx, func(q *database.Queries) error {
		var err error
		row, err = q.UpsertDecisionOutcomeWithTimestamp(ctx, database.UpsertDecisionOutcomeWithTimestampParams{
			DecisionID:     outcome.DecisionID,
			ChosenOptionID: outcome.ChosenOptionID,
			CommitmentNote: note,
			DecidedAt:      decidedAtTS,
		})
		if err != nil {
			return err
		}
		return q.TouchDecision(ctx, database.TouchDecisionParams{ID: outcome.DecisionID, UserID: userID})
	})
	if err != nil {
		return domain.DecisionOutcome{}, err
	}

	var noteStr *string
	if row.CommitmentNote.Valid {
		noteStr = &row.CommitmentNote.String
	}

	return domain.DecisionOutcome{
		DecisionID:     row.DecisionID,
		ChosenOptionID: row.ChosenOptionID,
		CommitmentNote: noteStr,
		DecidedAt:      row.DecidedAt.Time,
	}, nil
}

func (r *decisionRepository) UpsertEvaluation(ctx context.Context, eval domain.Evaluation, userID uuid.UUID) (domain.Evaluation, error) {
	if _, err := r.GetByID(ctx, eval.DecisionID, userID); err != nil {
		return domain.Evaluation{}, err
	}

	var createdAtTS pgtype.Timestamptz
	if !eval.CreatedAt.IsZero() {
		createdAtTS = pgtype.Timestamptz{Time: eval.CreatedAt, Valid: true}
	}

	var row database.Evaluation
	err := r.inTx(ctx, func(q *database.Queries) error {
		var err error
		row, err = q.UpsertEvaluation(ctx, database.UpsertEvaluationParams{
			DecisionID:        eval.DecisionID,
			SatisfactionAfter: pgtype.Int4{Int32: int32(eval.SatisfactionAfter), Valid: true},
			ConfidenceAfter:   pgtype.Int4{Int32: int32(eval.ConfidenceAfter), Valid: true},
			HindsightNotes:    pgtype.Text{String: eval.HindsightNotes, Valid: true},
			CreatedAt:         createdAtTS,
		})
		if err != nil {
			return err
		}
		return q.TouchDecision(ctx, database.TouchDecisionParams{ID: eval.DecisionID, UserID: userID})
	})
	if err != nil {
		return domain.Evaluation{}, err
	}

	return domain.Evaluation{
		ID:                row.ID,
		DecisionID:        row.DecisionID,
		SatisfactionAfter: int(row.SatisfactionAfter.Int32),
		ConfidenceAfter:   int(row.ConfidenceAfter.Int32),
		HindsightNotes:    row.HindsightNotes.String,
		CreatedAt:         row.CreatedAt.Time,
	}, nil
}

func (r *decisionRepository) AddTag(ctx context.Context, decisionID, tagID, userID uuid.UUID) error {
	if _, err := r.GetByID(ctx, decisionID, userID); err != nil {
		return err
	}

	// Verify tag belongs to user
	if _, err := r.q.GetTagByID(ctx, database.GetTagByIDParams{ID: tagID, UserID: userID}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return errors.New("tag not found")
		}
		return err
	}

	return r.inTx(ctx, func(q *database.Queries) error {
		if _, err := q.AddTagToDecision(ctx, database.AddTagToDecisionParams{
			DecisionID: decisionID,
			TagID:      tagID,
			UserID:     userID,
		}); err != nil {
			return err
		}
		return q.TouchDecision(ctx, database.TouchDecisionParams{ID: decisionID, UserID: userID})
	})
}

func (r *decisionRepository) RemoveTag(ctx context.Context, decisionID, tagID, userID uuid.UUID) error {
	return r.inTx(ctx, func(q *database.Queries) error {
		if err := q.RemoveTagFromDecision(ctx, database.RemoveTagFromDecisionParams{
			DecisionID: decisionID,
			TagID:      tagID,
			UserID:     userID,
		}); err != nil {
			return err
		}
		return q.TouchDecision(ctx, database.TouchDecisionParams{ID: decisionID, UserID: userID})
	})
}

// inTx runs fn in one transaction, so a write to a child row (option,
// outcome, evaluation, tag) and the decision's server_updated_at bump land
// together. Without the bump, GET /decisions?updated_since= misses the change.
func (r *decisionRepository) inTx(ctx context.Context, fn func(q *database.Queries) error) error {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if err := fn(r.q.WithTx(tx)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *decisionRepository) SyncDecision(ctx context.Context, id, userID uuid.UUID, data domain.SyncDecisionData) (domain.Decision, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return domain.Decision{}, err
	}
	defer tx.Rollback(ctx)

	qtx := r.q.WithTx(tx)

	// Fetch current decision directly (even if soft-deleted)
	rawRow, err := qtx.GetDecisionByID(ctx, database.GetDecisionByIDParams{ID: id, UserID: userID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Decision{}, domain.ErrDecisionNotFound
		}
		return domain.Decision{}, err
	}

	// Conflict check: if deleted or server has newer updated_at
	if rawRow.DeletedAt.Valid || rawRow.UpdatedAt.Time.After(data.LocalUpdatedAt) {
		serverDec, popErr := r.populateRelationsWithTx(ctx, qtx, rawRow)
		if popErr != nil {
			return domain.Decision{}, popErr
		}
		return domain.Decision{}, &domain.ErrConflict{ServerDecision: serverDec}
	}

	// Update decision attributes with client timestamp
	var contextText pgtype.Text
	if data.Context != nil {
		contextText = pgtype.Text{String: *data.Context, Valid: true}
	}
	var conf pgtype.Int4
	if data.ConfidenceBefore != nil {
		conf = pgtype.Int4{Int32: int32(*data.ConfidenceBefore), Valid: true}
	}
	var reviewDate pgtype.Date
	if data.ScheduledReviewDate != nil {
		reviewDate = pgtype.Date{Time: *data.ScheduledReviewDate, Valid: true}
	}

	_, err = qtx.UpdateDecisionWithLWW(ctx, database.UpdateDecisionWithLWWParams{
		ID:                  id,
		UserID:              userID,
		Title:               data.Title,
		Context:             contextText,
		ConfidenceBefore:    conf,
		ScheduledReviewDate: reviewDate,
		UpdatedAt:           pgtype.Timestamptz{Time: data.LocalUpdatedAt, Valid: true},
	})
	if err != nil {
		return domain.Decision{}, err
	}

	// Reconcile options. The order matters because of two constraints:
	// UNIQUE(decision_id, title), and decision_outcomes referencing the
	// chosen option.
	//  1. Delete the options left out of the request first, so a new option
	//     may reuse the title of one removed in the same sync. The currently
	//     chosen option waits until the outcome has moved off it.
	//  2. Update or create the options that were sent.
	//  3. Upsert the outcome.
	//  4. Delete the previously chosen option if it was left out and is no
	//     longer chosen.
	currentOpts, err := qtx.GetOptionsByDecisionID(ctx, id)
	if err != nil {
		return domain.Decision{}, err
	}

	var currentChosen uuid.UUID
	if outcomeRow, err := qtx.GetDecisionOutcome(ctx, id); err == nil {
		currentChosen = outcomeRow.ChosenOptionID
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return domain.Decision{}, err
	}

	sentIDs := make(map[uuid.UUID]bool)
	for _, opt := range data.Options {
		if opt.ID != nil && *opt.ID != uuid.Nil {
			sentIDs[*opt.ID] = true
		}
	}
	for _, co := range currentOpts {
		if !sentIDs[co.ID] && co.ID != currentChosen {
			if err := qtx.DeleteOptionByID(ctx, database.DeleteOptionByIDParams{ID: co.ID, DecisionID: id}); err != nil {
				return domain.Decision{}, err
			}
		}
	}

	clientKeyMap := make(map[string]uuid.UUID)
	keptOptionIDs := make(map[uuid.UUID]bool)

	for _, opt := range data.Options {
		var prosText pgtype.Text
		if opt.Pros != nil {
			prosText = pgtype.Text{String: *opt.Pros, Valid: true}
		}
		var consText pgtype.Text
		if opt.Cons != nil {
			consText = pgtype.Text{String: *opt.Cons, Valid: true}
		}

		var savedID uuid.UUID
		if opt.ID != nil && *opt.ID != uuid.Nil {
			updatedOpt, err := qtx.UpdateOption(ctx, database.UpdateOptionParams{
				ID:         *opt.ID,
				DecisionID: id,
				Title:      opt.Title,
				Pros:       prosText,
				Cons:       consText,
			})
			if err == nil {
				savedID = updatedOpt.ID
			} else if !errors.Is(err, pgx.ErrNoRows) {
				return domain.Decision{}, err
			}
			// ErrNoRows: this decision has no option with that id (e.g. it was
			// deleted by another device), so it is created again below.
		}
		if savedID == uuid.Nil {
			createdOpt, err := qtx.CreateOption(ctx, database.CreateOptionParams{
				DecisionID: id,
				Title:      opt.Title,
				Pros:       prosText,
				Cons:       consText,
			})
			if err != nil {
				return domain.Decision{}, err
			}
			savedID = createdOpt.ID
		}

		keptOptionIDs[savedID] = true
		if opt.ClientKey != nil && *opt.ClientKey != "" {
			clientKeyMap[*opt.ClientKey] = savedID
		}
	}

	// Reconcile outcome if provided. Committing is final, so a request
	// without an outcome leaves the server's outcome as it is.
	finalChosen := currentChosen
	if data.Outcome != nil {
		var chosenID uuid.UUID
		if data.Outcome.ChosenOptionID != nil && keptOptionIDs[*data.Outcome.ChosenOptionID] {
			chosenID = *data.Outcome.ChosenOptionID
		} else if data.Outcome.ChosenOptionKey != nil && *data.Outcome.ChosenOptionKey != "" {
			chosenID = clientKeyMap[*data.Outcome.ChosenOptionKey]
		}
		if chosenID == uuid.Nil {
			// Saving the rest without the commitment would silently lose it.
			return domain.Decision{}, domain.ErrInvalidChosenOption
		}

		var note pgtype.Text
		if data.Outcome.CommitmentNote != nil {
			note = pgtype.Text{String: *data.Outcome.CommitmentNote, Valid: true}
		}
		var decidedAtTS pgtype.Timestamptz
		if data.Outcome.DecidedAt != nil && !data.Outcome.DecidedAt.IsZero() {
			decidedAtTS = pgtype.Timestamptz{Time: *data.Outcome.DecidedAt, Valid: true}
		}

		_, err = qtx.UpsertDecisionOutcomeWithTimestamp(ctx, database.UpsertDecisionOutcomeWithTimestampParams{
			DecisionID:     id,
			ChosenOptionID: chosenID,
			CommitmentNote: note,
			DecidedAt:      decidedAtTS,
		})
		if err != nil {
			return domain.Decision{}, err
		}
		finalChosen = chosenID
	}

	if currentChosen != uuid.Nil && !keptOptionIDs[currentChosen] && finalChosen != currentChosen {
		if err := qtx.DeleteOptionByID(ctx, database.DeleteOptionByIDParams{ID: currentChosen, DecisionID: id}); err != nil {
			return domain.Decision{}, err
		}
	}

	// Reconcile tags
	if err := qtx.ClearTagsForDecision(ctx, id); err != nil {
		return domain.Decision{}, err
	}
	for _, tagID := range data.TagIDs {
		_, err := qtx.AddTagToDecision(ctx, database.AddTagToDecisionParams{
			DecisionID: id,
			TagID:      tagID,
			UserID:     userID,
		})
		if err != nil {
			return domain.Decision{}, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.Decision{}, err
	}

	return r.GetByID(ctx, id, userID)
}

// populateRelations assembles nested options, outcome, evaluation, and tags for a decision.
func (r *decisionRepository) populateRelations(ctx context.Context, row database.Decision) (domain.Decision, error) {
	return r.populateRelationsWithTx(ctx, r.q, row)
}

func (r *decisionRepository) populateRelationsWithTx(ctx context.Context, q *database.Queries, row database.Decision) (domain.Decision, error) {
	id := row.ID
	optRows, err := q.GetOptionsByDecisionID(ctx, id)
	if err != nil {
		return domain.Decision{}, err
	}

	options := make([]domain.Option, len(optRows))
	for i, o := range optRows {
		var pStr, cStr *string
		if o.Pros.Valid {
			pStr = &o.Pros.String
		}
		if o.Cons.Valid {
			cStr = &o.Cons.String
		}
		options[i] = domain.Option{
			ID:         o.ID,
			DecisionID: o.DecisionID,
			Title:      o.Title,
			Pros:       pStr,
			Cons:       cStr,
			CreatedAt:  o.CreatedAt.Time,
		}
	}

	var outcome *domain.DecisionOutcome
	outcomeRow, err := q.GetDecisionOutcome(ctx, id)
	if err == nil {
		var note *string
		if outcomeRow.CommitmentNote.Valid {
			note = &outcomeRow.CommitmentNote.String
		}
		outcome = &domain.DecisionOutcome{
			DecisionID:     outcomeRow.DecisionID,
			ChosenOptionID: outcomeRow.ChosenOptionID,
			CommitmentNote: note,
			DecidedAt:      outcomeRow.DecidedAt.Time,
		}
	}

	var eval *domain.Evaluation
	evalRow, err := q.GetEvaluationByDecisionID(ctx, id)
	if err == nil {
		eval = &domain.Evaluation{
			ID:                evalRow.ID,
			DecisionID:        evalRow.DecisionID,
			SatisfactionAfter: int(evalRow.SatisfactionAfter.Int32),
			ConfidenceAfter:   int(evalRow.ConfidenceAfter.Int32),
			HindsightNotes:    evalRow.HindsightNotes.String,
			CreatedAt:         evalRow.CreatedAt.Time,
		}
	}

	tagRows, err := q.GetTagsForDecision(ctx, id)
	if err != nil {
		return domain.Decision{}, err
	}
	tags := make([]domain.Tag, len(tagRows))
	for i, t := range tagRows {
		tags[i] = domain.Tag{
			ID:       t.ID,
			Name:     t.Name,
			ColorHex: t.ColorHex.String,
		}
	}

	var ctxStr *string
	if row.Context.Valid {
		ctxStr = &row.Context.String
	}
	var confInt *int
	if row.ConfidenceBefore.Valid {
		v := int(row.ConfidenceBefore.Int32)
		confInt = &v
	}
	var reviewDateTime *time.Time
	if row.ScheduledReviewDate.Valid {
		t := row.ScheduledReviewDate.Time
		reviewDateTime = &t
	}
	var delTime *time.Time
	if row.DeletedAt.Valid {
		t := row.DeletedAt.Time
		delTime = &t
	}

	return domain.Decision{
		ID:                  row.ID,
		UserID:              row.UserID,
		Title:               row.Title,
		Context:             ctxStr,
		ConfidenceBefore:    confInt,
		ScheduledReviewDate: reviewDateTime,
		CreatedAt:           row.CreatedAt.Time,
		UpdatedAt:           row.UpdatedAt.Time,
		ServerUpdatedAt:     row.ServerUpdatedAt.Time,
		DeletedAt:           delTime,
		Options:             options,
		Outcome:             outcome,
		Evaluation:          eval,
		Tags:                tags,
	}, nil
}
