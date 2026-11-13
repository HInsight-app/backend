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

	createdRow, err := qtx.CreateDecision(ctx, database.CreateDecisionParams{
		UserID:              d.UserID,
		Title:               d.Title,
		Context:             contextText,
		ConfidenceBefore:    conf,
		ScheduledReviewDate: reviewDate,
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
			return domain.Decision{}, errors.New("decision not found")
		}
		return domain.Decision{}, err
	}

	if row.DeletedAt.Valid {
		return domain.Decision{}, errors.New("decision not found")
	}

	return r.populateRelations(ctx, row.ID, row.UserID, row.Title, row.Context, row.ConfidenceBefore, row.ScheduledReviewDate, row.CreatedAt, row.UpdatedAt, row.DeletedAt)
}

func (r *decisionRepository) GetActiveByUserID(ctx context.Context, userID uuid.UUID) ([]domain.Decision, error) {
	rows, err := r.q.GetActiveDecisionsByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	decisions := make([]domain.Decision, 0, len(rows))
	for _, row := range rows {
		d, err := r.populateRelations(ctx, row.ID, row.UserID, row.Title, row.Context, row.ConfidenceBefore, row.ScheduledReviewDate, row.CreatedAt, row.UpdatedAt, pgtype.Timestamptz{Valid: false})
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
			current, getErr := r.GetByID(ctx, d.ID, d.UserID)
			if getErr != nil {
				return domain.Decision{}, getErr
			}
			return domain.Decision{}, &domain.ErrConflict{ServerDecision: current}
		}
		return domain.Decision{}, err
	}

	return r.populateRelations(ctx, updatedRow.ID, updatedRow.UserID, updatedRow.Title, updatedRow.Context, updatedRow.ConfidenceBefore, updatedRow.ScheduledReviewDate, updatedRow.CreatedAt, updatedRow.UpdatedAt, pgtype.Timestamptz{Valid: false})
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
		current, getErr := r.GetByID(ctx, id, userID)
		if getErr != nil {
			return getErr
		}
		return &domain.ErrConflict{ServerDecision: current}
	}

	return nil
}

func (r *decisionRepository) CreateOption(ctx context.Context, opt domain.Option) (domain.Option, error) {
	var prosText pgtype.Text
	if opt.Pros != nil {
		prosText = pgtype.Text{String: *opt.Pros, Valid: true}
	}
	var consText pgtype.Text
	if opt.Cons != nil {
		consText = pgtype.Text{String: *opt.Cons, Valid: true}
	}

	row, err := r.q.CreateOption(ctx, database.CreateOptionParams{
		DecisionID: opt.DecisionID,
		Title:      opt.Title,
		Pros:       prosText,
		Cons:       consText,
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

	return r.q.DeleteOptionByID(ctx, database.DeleteOptionByIDParams{
		ID:         optionID,
		DecisionID: decisionID,
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

	row, err := r.q.UpsertDecisionOutcome(ctx, database.UpsertDecisionOutcomeParams{
		DecisionID:     outcome.DecisionID,
		ChosenOptionID: outcome.ChosenOptionID,
		CommitmentNote: note,
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

	row, err := r.q.UpsertEvaluation(ctx, database.UpsertEvaluationParams{
		DecisionID:        eval.DecisionID,
		SatisfactionAfter: pgtype.Int4{Int32: int32(eval.SatisfactionAfter), Valid: true},
		ConfidenceAfter:   pgtype.Int4{Int32: int32(eval.ConfidenceAfter), Valid: true},
		HindsightNotes:    pgtype.Text{String: eval.HindsightNotes, Valid: true},
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

	_, err := r.q.AddTagToDecision(ctx, database.AddTagToDecisionParams{
		DecisionID: decisionID,
		TagID:      tagID,
		UserID:     userID,
	})
	return err
}

func (r *decisionRepository) RemoveTag(ctx context.Context, decisionID, tagID, userID uuid.UUID) error {
	return r.q.RemoveTagFromDecision(ctx, database.RemoveTagFromDecisionParams{
		DecisionID: decisionID,
		TagID:      tagID,
		UserID:     userID,
	})
}

// populateRelations assembles nested options, outcome, evaluation, and tags for a decision.
func (r *decisionRepository) populateRelations(
	ctx context.Context,
	id, userID uuid.UUID,
	title string,
	ctxText pgtype.Text,
	conf pgtype.Int4,
	rDate pgtype.Date,
	cAt, uAt, dAt pgtype.Timestamptz,
) (domain.Decision, error) {
	optRows, err := r.q.GetOptionsByDecisionID(ctx, id)
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
	outcomeRow, err := r.q.GetDecisionOutcome(ctx, id)
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
	evalRow, err := r.q.GetEvaluationByDecisionID(ctx, id)
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

	tagRows, err := r.q.GetTagsForDecision(ctx, id)
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
	if ctxText.Valid {
		ctxStr = &ctxText.String
	}
	var confInt *int
	if conf.Valid {
		v := int(conf.Int32)
		confInt = &v
	}
	var reviewDateTime *time.Time
	if rDate.Valid {
		t := rDate.Time
		reviewDateTime = &t
	}
	var delTime *time.Time
	if dAt.Valid {
		t := dAt.Time
		delTime = &t
	}

	return domain.Decision{
		ID:                  id,
		UserID:              userID,
		Title:               title,
		Context:             ctxStr,
		ConfidenceBefore:    confInt,
		ScheduledReviewDate: reviewDateTime,
		CreatedAt:           cAt.Time,
		UpdatedAt:           uAt.Time,
		DeletedAt:           delTime,
		Options:             options,
		Outcome:             outcome,
		Evaluation:          eval,
		Tags:                tags,
	}, nil
}
