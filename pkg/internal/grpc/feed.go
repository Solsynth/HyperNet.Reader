package grpc

import (
	"context"
	"time"

	iproto "git.solsynth.dev/hypernet/interactive/pkg/proto"
	"git.solsynth.dev/hypernet/nexus/pkg/nex"
	"git.solsynth.dev/hypernet/reader/pkg/internal/models"
	"git.solsynth.dev/hypernet/reader/pkg/internal/services"
	"github.com/samber/lo"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (v *Server) GetFeed(_ context.Context, in *iproto.GetFeedRequest) (*iproto.GetFeedResponse, error) {
	limit := int(in.GetLimit())
	var cursor *time.Time
	if in.Cursor != nil {
		cursor = lo.ToPtr(time.UnixMilli(int64(in.GetCursor())))
	}
	articles, err := services.GetTodayFeedRandomly(limit, cursor)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &iproto.GetFeedResponse{
		Items: lo.Map(articles, func(item models.SubscriptionItem, _ int) *iproto.FeedItem {
			return &iproto.FeedItem{
				Type:      "reader.feed",
				Content:   nex.EncodeMap(item),
				CreatedAt: uint64(item.CreatedAt.Unix()),
			}
		}),
	}, nil
}
