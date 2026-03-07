package handler

import (
	"context"
	"errors"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	pb "github.com/chaowen/budget/gen/budget/v1"
	"github.com/chaowen/budget/internal/middleware"
	"github.com/chaowen/budget/internal/model"
	"github.com/chaowen/budget/internal/repository"
)

type AssetHandler struct {
	pb.UnimplementedAssetServiceServer
	assetRepo      *repository.AssetRepository
	accountingRepo *repository.AccountingRepository
	userRepo       *repository.UserRepository
	currencyRepo   *repository.CurrencyRepository
}

func NewAssetHandler(assetRepo *repository.AssetRepository, accountingRepo *repository.AccountingRepository, userRepo *repository.UserRepository, currencyRepo *repository.CurrencyRepository) *AssetHandler {
	return &AssetHandler{
		assetRepo:      assetRepo,
		accountingRepo: accountingRepo,
		userRepo:       userRepo,
		currencyRepo:   currencyRepo,
	}
}

func (h *AssetHandler) ListAssetTypes(ctx context.Context, req *pb.ListAssetTypesRequest) (*pb.ListAssetTypesResponse, error) {
	var category *model.AssetCategory
	if req.Category != pb.AssetCategory_ASSET_CATEGORY_UNSPECIFIED {
		c := protoToAssetCategory(req.Category)
		category = &c
	}

	types, err := h.assetRepo.ListAssetTypes(ctx, category)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list asset types")
	}

	pbTypes := make([]*pb.AssetType, len(types))
	for i, t := range types {
		pbTypes[i] = &pb.AssetType{
			Id:       t.ID.String(),
			Name:     t.Name,
			Category: assetCategoryToProto(t.Category),
		}
	}

	return &pb.ListAssetTypesResponse{
		AssetTypes: pbTypes,
	}, nil
}

func (h *AssetHandler) CreateAsset(ctx context.Context, req *pb.CreateAssetRequest) (*pb.CreateAssetResponse, error) {
	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		return nil, err
	}

	if req.AssetTypeId == "" {
		return nil, status.Error(codes.InvalidArgument, "asset_type_id is required")
	}
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}

	assetTypeID, err := uuid.Parse(req.AssetTypeId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid asset_type_id")
	}

	currentValue := decimal.Zero
	if req.CurrentValue != "" {
		currentValue, err = decimal.NewFromString(req.CurrentValue)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid current_value")
		}
	}

	currency := strings.ToUpper(strings.TrimSpace(req.Currency))
	if currency == "" {
		user, err := h.userRepo.GetByID(ctx, userID)
		if err != nil {
			return nil, status.Error(codes.Internal, "failed to load user profile")
		}
		currency = user.BaseCurrency
	}

	var customFields []byte
	if req.CustomFields != nil {
		customFields, _ = req.CustomFields.MarshalJSON()
	}

	asset := &model.Asset{
		UserID:       userID,
		AssetTypeID:  assetTypeID,
		Name:         req.Name,
		Currency:     currency,
		CurrentValue: currentValue,
		IsLiability:  req.IsLiability,
		CustomFields: customFields,
	}

	if err := h.assetRepo.Create(ctx, asset); err != nil {
		return nil, status.Error(codes.Internal, "failed to create asset")
	}
	if _, err := h.accountingRepo.EnsureAssetAccount(ctx, asset); err != nil {
		return nil, status.Error(codes.Internal, "failed to create asset ledger account")
	}

	_ = h.assetRepo.RecordSnapshot(ctx, &model.AssetSnapshot{
		AssetID: asset.ID,
		Value:   asset.CurrentValue,
	})

	// Refetch to get asset type info
	asset, _ = h.assetRepo.GetByID(ctx, asset.ID, userID)

	return &pb.CreateAssetResponse{
		Asset: assetToProto(asset),
	}, nil
}

func (h *AssetHandler) GetAsset(ctx context.Context, req *pb.GetAssetRequest) (*pb.GetAssetResponse, error) {
	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		return nil, err
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid asset ID")
	}

	asset, err := h.assetRepo.GetByID(ctx, id, userID)
	if err != nil {
		if errors.Is(err, repository.ErrAssetNotFound) {
			return nil, status.Error(codes.NotFound, "asset not found")
		}
		return nil, status.Error(codes.Internal, "failed to get asset")
	}

	return &pb.GetAssetResponse{
		Asset: assetToProto(asset),
	}, nil
}

func (h *AssetHandler) ListAssets(ctx context.Context, req *pb.ListAssetsRequest) (*pb.ListAssetsResponse, error) {
	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		return nil, err
	}

	var category *model.AssetCategory
	if req.Category != pb.AssetCategory_ASSET_CATEGORY_UNSPECIFIED {
		c := protoToAssetCategory(req.Category)
		category = &c
	}

	assets, err := h.assetRepo.List(ctx, userID, category, req.IncludeLiabilities)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list assets")
	}

	user, err := h.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to load user profile")
	}

	allAssets, err := h.assetRepo.List(ctx, userID, nil, true)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list assets for totals")
	}

	totalAssets := decimal.Zero
	totalLiabilities := decimal.Zero
	for _, asset := range allAssets {
		converted, convErr := h.currencyRepo.ConvertAmount(ctx, asset.CurrentValue, asset.Currency, user.BaseCurrency)
		if convErr != nil {
			if errors.Is(convErr, repository.ErrExchangeRateNotFound) {
				return nil, status.Error(codes.FailedPrecondition, "missing exchange rate for asset totals")
			}
			return nil, status.Error(codes.Internal, "failed to convert asset totals")
		}

		if asset.IsLiability {
			totalLiabilities = totalLiabilities.Add(converted)
		} else {
			totalAssets = totalAssets.Add(converted)
		}
	}
	netWorth := totalAssets.Sub(totalLiabilities)

	pbAssets := make([]*pb.Asset, len(assets))
	for i, a := range assets {
		pbAssets[i] = assetToProto(&a)
	}

	return &pb.ListAssetsResponse{
		Assets:                pbAssets,
		TotalAssetsValue:      totalAssets.String(),
		TotalLiabilitiesValue: totalLiabilities.String(),
		NetWorth:              netWorth.String(),
		BaseCurrency:          user.BaseCurrency,
	}, nil
}

func (h *AssetHandler) UpdateAsset(ctx context.Context, req *pb.UpdateAssetRequest) (*pb.UpdateAssetResponse, error) {
	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		return nil, err
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid asset ID")
	}

	asset, err := h.assetRepo.GetByID(ctx, id, userID)
	if err != nil {
		if errors.Is(err, repository.ErrAssetNotFound) {
			return nil, status.Error(codes.NotFound, "asset not found")
		}
		return nil, status.Error(codes.Internal, "failed to get asset")
	}

	if req.Name != "" {
		asset.Name = req.Name
	}

	if req.CurrentValue != "" {
		currentValue, err := decimal.NewFromString(req.CurrentValue)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid current_value")
		}
		if err := h.accountingRepo.AdjustAssetToValue(ctx, userID, asset, currentValue, "manual asset update"); err != nil {
			return nil, status.Error(codes.Internal, "failed to adjust asset ledger balance")
		}
		asset.CurrentValue = currentValue
	}

	if req.CustomFields != nil {
		customFields, _ := req.CustomFields.MarshalJSON()
		asset.CustomFields = customFields
	}

	if err := h.assetRepo.Update(ctx, asset); err != nil {
		return nil, status.Error(codes.Internal, "failed to update asset")
	}

	_ = h.assetRepo.RecordSnapshot(ctx, &model.AssetSnapshot{
		AssetID: asset.ID,
		Value:   asset.CurrentValue,
	})

	return &pb.UpdateAssetResponse{
		Asset: assetToProto(asset),
	}, nil
}

func (h *AssetHandler) DeleteAsset(ctx context.Context, req *pb.DeleteAssetRequest) (*pb.DeleteAssetResponse, error) {
	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		return nil, err
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid asset ID")
	}

	if err := h.assetRepo.Delete(ctx, id, userID); err != nil {
		if errors.Is(err, repository.ErrAssetNotFound) {
			return nil, status.Error(codes.NotFound, "asset not found")
		}
		if errors.Is(err, repository.ErrAssetInUse) {
			blockers, blockerErr := h.assetRepo.ListDeleteBlockers(ctx, id, userID, 3)
			if blockerErr != nil || len(blockers) == 0 {
				return nil, status.Error(codes.FailedPrecondition, "cannot delete asset with linked transactions or transfers")
			}

			parts := make([]string, 0, len(blockers))
			for _, b := range blockers {
				parts = append(parts, b.Kind+" "+b.ReferenceID.String()+" ("+b.Description+")")
			}

			return nil, status.Error(codes.FailedPrecondition, "cannot delete asset; linked records: "+strings.Join(parts, "; "))
		}
		return nil, status.Error(codes.Internal, "failed to delete asset")
	}

	return &pb.DeleteAssetResponse{}, nil
}

func (h *AssetHandler) RecordAssetSnapshot(ctx context.Context, req *pb.RecordAssetSnapshotRequest) (*pb.RecordAssetSnapshotResponse, error) {
	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		return nil, err
	}

	assetID, err := uuid.Parse(req.AssetId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid asset_id")
	}

	// Verify asset belongs to user
	_, err = h.assetRepo.GetByID(ctx, assetID, userID)
	if err != nil {
		if errors.Is(err, repository.ErrAssetNotFound) {
			return nil, status.Error(codes.NotFound, "asset not found")
		}
		return nil, status.Error(codes.Internal, "failed to get asset")
	}

	value, err := decimal.NewFromString(req.Value)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid value")
	}

	snapshot := &model.AssetSnapshot{
		AssetID: assetID,
		Value:   value,
	}

	if req.RecordedAt != nil {
		snapshot.RecordedAt = req.RecordedAt.AsTime()
	}

	if err := h.assetRepo.RecordSnapshot(ctx, snapshot); err != nil {
		return nil, status.Error(codes.Internal, "failed to record snapshot")
	}

	return &pb.RecordAssetSnapshotResponse{
		Snapshot: &pb.AssetSnapshot{
			Id:         snapshot.ID.String(),
			AssetId:    snapshot.AssetID.String(),
			Value:      snapshot.Value.String(),
			RecordedAt: timestamppb.New(snapshot.RecordedAt),
		},
	}, nil
}

func (h *AssetHandler) GetAssetHistory(ctx context.Context, req *pb.GetAssetHistoryRequest) (*pb.GetAssetHistoryResponse, error) {
	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		return nil, err
	}

	assetID, err := uuid.Parse(req.AssetId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid asset_id")
	}

	// Verify asset belongs to user
	_, err = h.assetRepo.GetByID(ctx, assetID, userID)
	if err != nil {
		if errors.Is(err, repository.ErrAssetNotFound) {
			return nil, status.Error(codes.NotFound, "asset not found")
		}
		return nil, status.Error(codes.Internal, "failed to get asset")
	}

	snapshots, err := h.assetRepo.GetSnapshots(ctx, assetID, nil, nil)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get asset history")
	}

	pbSnapshots := make([]*pb.AssetSnapshot, len(snapshots))
	for i, s := range snapshots {
		pbSnapshots[i] = &pb.AssetSnapshot{
			Id:         s.ID.String(),
			AssetId:    s.AssetID.String(),
			Value:      s.Value.String(),
			RecordedAt: timestamppb.New(s.RecordedAt),
		}
	}

	return &pb.GetAssetHistoryResponse{
		Snapshots: pbSnapshots,
	}, nil
}

func assetToProto(a *model.Asset) *pb.Asset {
	var customFields *structpb.Struct
	if len(a.CustomFields) > 0 {
		customFields = &structpb.Struct{}
		_ = customFields.UnmarshalJSON(a.CustomFields)
	}

	return &pb.Asset{
		Id:            a.ID.String(),
		AssetTypeId:   a.AssetTypeID.String(),
		AssetTypeName: a.AssetTypeName,
		Category:      assetCategoryToProto(a.Category),
		Name:          a.Name,
		Currency:      a.Currency,
		CurrentValue:  a.CurrentValue.String(),
		IsLiability:   a.IsLiability,
		CustomFields:  customFields,
		CreatedAt:     timestamppb.New(a.CreatedAt),
		UpdatedAt:     timestamppb.New(a.UpdatedAt),
	}
}

func assetCategoryToProto(c model.AssetCategory) pb.AssetCategory {
	switch c {
	case model.AssetCategoryCash:
		return pb.AssetCategory_ASSET_CATEGORY_CASH
	case model.AssetCategoryBank:
		return pb.AssetCategory_ASSET_CATEGORY_BANK
	case model.AssetCategoryInvestment:
		return pb.AssetCategory_ASSET_CATEGORY_INVESTMENT
	case model.AssetCategoryRetirement:
		return pb.AssetCategory_ASSET_CATEGORY_RETIREMENT
	case model.AssetCategoryProperty:
		return pb.AssetCategory_ASSET_CATEGORY_PROPERTY
	case model.AssetCategoryLiability:
		return pb.AssetCategory_ASSET_CATEGORY_LIABILITY
	case model.AssetCategoryCustom:
		return pb.AssetCategory_ASSET_CATEGORY_CUSTOM
	default:
		return pb.AssetCategory_ASSET_CATEGORY_UNSPECIFIED
	}
}

func protoToAssetCategory(c pb.AssetCategory) model.AssetCategory {
	switch c {
	case pb.AssetCategory_ASSET_CATEGORY_CASH:
		return model.AssetCategoryCash
	case pb.AssetCategory_ASSET_CATEGORY_BANK:
		return model.AssetCategoryBank
	case pb.AssetCategory_ASSET_CATEGORY_INVESTMENT:
		return model.AssetCategoryInvestment
	case pb.AssetCategory_ASSET_CATEGORY_RETIREMENT:
		return model.AssetCategoryRetirement
	case pb.AssetCategory_ASSET_CATEGORY_PROPERTY:
		return model.AssetCategoryProperty
	case pb.AssetCategory_ASSET_CATEGORY_LIABILITY:
		return model.AssetCategoryLiability
	case pb.AssetCategory_ASSET_CATEGORY_CUSTOM:
		return model.AssetCategoryCustom
	default:
		return model.AssetCategoryCustom
	}
}
