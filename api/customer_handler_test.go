package api

import (
	"context"
	"fmt"
	"testing"

	"github.com/choirulanwar/simple-bank/db/sqlc"
	"github.com/choirulanwar/simple-bank/internal/mock"
	"github.com/choirulanwar/simple-bank/internal/repository"
	"github.com/choirulanwar/simple-bank/api/pb"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestCustomerHandler_CreateCustomer(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mock.NewMockQuerier(ctrl)

	t.Run("Success", func(t *testing.T) {
		expectedCustomer := sqlc.Customer{
			ID:           1,
			Name:         "John Doe",
			Email:        "john@example.com",
			PasswordHash: "hashed",
			IsActive:     true,
			CreatedAt:    timestamppb.Now().AsTime(),
			UpdatedAt:    timestamppb.Now().AsTime(),
		}

		mockQuerier.EXPECT().
			GetCustomerByEmail(gomock.Any(), "john@example.com").
			Return(sqlc.Customer{}, nil)

		mockQuerier.EXPECT().
			CreateCustomer(gomock.Any(), gomock.Any()).
			Return(expectedCustomer, nil)

		resp, err := NewCustomerHandler(repository.NewCustomerRepo(mockQuerier)).
			CreateCustomer(context.Background(), &pb.CreateCustomerRequest{
				Name:     "John Doe",
				Email:    "john@example.com",
				Password: "SecureP@ss1",
			})

		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Equal(t, expectedCustomer.ID, resp.Customer.Id)
		require.Equal(t, "John Doe", resp.Customer.Name)
	})

	t.Run("Duplicate Email", func(t *testing.T) {
		mockQuerier.EXPECT().
			GetCustomerByEmail(gomock.Any(), "existing@example.com").
			Return(sqlc.Customer{}, nil)

		mockQuerier.EXPECT().
			CreateCustomer(gomock.Any(), gomock.Any()).
			Return(sqlc.Customer{}, fmt.Errorf("email already registered"))

		_, err := NewCustomerHandler(repository.NewCustomerRepo(mockQuerier)).
			CreateCustomer(context.Background(), &pb.CreateCustomerRequest{
				Name:     "Jane",
				Email:    "existing@example.com",
				Password: "SecureP@ss1",
			})

		require.Error(t, err)
		st := status.Convert(err)
		require.Equal(t, codes.AlreadyExists, st.Code())
	})

	t.Run("Invalid Email", func(t *testing.T) {
		// Handler validates email before calling repo, so no mock expectations needed
		_, err := NewCustomerHandler(repository.NewCustomerRepo(mockQuerier)).
			CreateCustomer(context.Background(), &pb.CreateCustomerRequest{
				Name:     "Jane",
				Email:    "invalid-email",
				Password: "SecureP@ss1",
			})

		require.Error(t, err)
		st := status.Convert(err)
		require.Equal(t, codes.InvalidArgument, st.Code())
	})
}

func TestCustomerHandler_GetCustomer(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mock.NewMockQuerier(ctrl)

	t.Run("Success", func(t *testing.T) {
		customer := sqlc.Customer{
			ID:        1,
			Name:      "John Doe",
			Email:     "john@example.com",
			IsActive:  true,
			CreatedAt: timestamppb.Now().AsTime(),
			UpdatedAt: timestamppb.Now().AsTime(),
		}

		mockQuerier.EXPECT().
			GetCustomer(gomock.Any(), int64(1)).
			Return(customer, nil)

		resp, err := NewCustomerHandler(repository.NewCustomerRepo(mockQuerier)).
			GetCustomer(context.Background(), &pb.GetCustomerRequest{Id: 1})

		require.NoError(t, err)
		require.Equal(t, customer.ID, resp.Customer.Id)
	})

t.Run("Not Found", func(t *testing.T) {
		mockQuerier.EXPECT().
			GetCustomer(gomock.Any(), int64(999)).
			Return(sqlc.Customer{}, fmt.Errorf("customer not found"))

		_, err := NewCustomerHandler(repository.NewCustomerRepo(mockQuerier)).
			GetCustomer(context.Background(), &pb.GetCustomerRequest{Id: 999})
		require.Error(t, err)
		st := status.Convert(err)
		require.Equal(t, codes.NotFound, st.Code())
	})
}

func TestCustomerHandler_ListCustomers(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mock.NewMockQuerier(ctrl)
	handler := NewCustomerHandler(repository.NewCustomerRepo(mockQuerier))

	t.Run("Success", func(t *testing.T) {
		customers := []sqlc.Customer{
			{ID: 1, Name: "A", Email: "a@a.com", IsActive: true},
			{ID: 2, Name: "B", Email: "b@b.com", IsActive: true},
		}

		mockQuerier.EXPECT().
			ListCustomers(gomock.Any(), sqlc.ListCustomersParams{
				Limit:  10,
				Offset: 0,
			}).Return(customers, nil)

		resp, err := handler.ListCustomers(context.Background(), &pb.ListCustomersRequest{Limit: 10, Offset: 0})

		require.NoError(t, err)
		require.Len(t, resp.Customers, 2)
	})
}