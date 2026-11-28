package persistence

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/text/cases"
	"taotale/backend/internal/persistence/queries"
)

type TopicPalette struct {
	LightBackground, LightText, DarkBackground, DarkText string
}

type TopicRecord struct {
	queries.Topic
	Usage queries.CountTopicUsageRow
}

func (s Library) SaveTopic(ctx context.Context, id pgtype.UUID, name *string, colors *TopicPalette) (TopicRecord, error) {
	var result TopicRecord
	err := pgx.BeginFunc(ctx, s.Pool, func(tx pgx.Tx) error {
		q := queries.New(tx)
		owner, err := q.LockOwner(ctx)
		if err != nil {
			return err
		}
		var current queries.Topic
		if id.Valid {
			current, err = q.GetTopic(ctx, id)
			if err != nil {
				return missing(err, ErrTopicNotFound)
			}
		}
		nextName, nextKey := current.Name, current.NameKey
		if name != nil {
			nextName, nextKey = *name, cases.Fold().String(*name)
			existing, err := q.FindTopicByName(ctx, queries.FindTopicByNameParams{UserID: owner, NameKey: nextKey})
			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				return err
			}
			if err == nil && existing != id {
				return &DuplicateTopic{existing}
			}
		}
		palette := valueOr(colors, TopicPalette{current.LightBackground, current.LightText, current.DarkBackground, current.DarkText})
		if !id.Valid {
			result.Topic, err = q.InsertTopic(ctx, queries.InsertTopicParams{
				UserID: owner, Name: nextName, NameKey: nextKey, LightBackground: palette.LightBackground,
				LightText: palette.LightText, DarkBackground: palette.DarkBackground, DarkText: palette.DarkText,
			})
			return err
		}
		result.Topic, err = q.UpdateTopic(ctx, queries.UpdateTopicParams{
			ID: id, Name: nextName, NameKey: nextKey, LightBackground: palette.LightBackground,
			LightText: palette.LightText, DarkBackground: palette.DarkBackground, DarkText: palette.DarkText,
		})
		if err == nil {
			result.Usage, err = q.CountTopicUsage(ctx, id)
		}
		return err
	})
	return result, err
}

func (s Library) DeleteTopic(ctx context.Context, id pgtype.UUID) error {
	return pgx.BeginFunc(ctx, s.Pool, func(tx pgx.Tx) error {
		q := queries.New(tx)
		if _, err := q.LockOwner(ctx); err != nil {
			return err
		}
		topic, err := q.GetTopic(ctx, id)
		if err != nil {
			return missing(err, ErrTopicNotFound)
		}
		if topic.SystemKey.Valid {
			return ErrDefaultTopic
		}
		return q.DeleteCustomTopic(ctx, id)
	})
}
