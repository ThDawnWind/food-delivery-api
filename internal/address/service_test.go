package address

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
)

type fakeRepository struct {
	createInput   *CreateAddress
	createAddress *Address
	createErr     error

	getByIDAddress *Address
	getByIDErr     error
	getByIDID      int64
	getByIDUserID  int64

	listByUserAddresses []Address
	listByUserErr       error
	listByUserID        int64

	deleteAddressID int64
	deleteUserID    int64
	deleteErr       error

	updateAddressID int64
	updateUserID    int64
	updateInput     *UpdateAddress
	updateAddress   *Address
	updateErr       error
}

func (f *fakeRepository) Create(ctx context.Context, input *CreateAddress) (*Address, error) {
	f.createInput = input

	if f.createErr != nil {
		return nil, f.createErr
	}

	return f.createAddress, nil
}

func (f *fakeRepository) GetByID(ctx context.Context, addressID int64, userID int64) (*Address, error) {
	f.getByIDID = addressID
	f.getByIDUserID = userID

	if f.getByIDErr != nil {
		return nil, f.getByIDErr
	}

	return f.getByIDAddress, nil
}

func (f *fakeRepository) ListByUser(ctx context.Context, userID int64) ([]Address, error) {
	f.listByUserID = userID

	if f.listByUserErr != nil {
		return nil, f.listByUserErr
	}

	return f.listByUserAddresses, nil
}

func (f *fakeRepository) Update(ctx context.Context, addressID int64, userID int64, input *UpdateAddress) (*Address, error) {
	f.updateAddressID = addressID
	f.updateUserID = userID
	f.updateInput = input

	if f.updateErr != nil {
		return nil, f.updateErr
	}

	return f.updateAddress, nil
}

func (f *fakeRepository) Delete(ctx context.Context, addressID int64, userID int64) error {
	f.deleteAddressID = addressID
	f.deleteUserID = userID

	return f.deleteErr
}

func TestService_Create(t *testing.T) {
	repository := &fakeRepository{
		createAddress: &Address{
			ID:          1,
			UserID:      10,
			City:        "Amsterdam",
			Street:      "Test Street",
			HouseNumber: "10",
		},
	}

	service := NewService(repository)

	address, err := service.Create(
		context.Background(),
		&CreateAddress{
			UserID:      10,
			City:        "  Amsterdam  ",
			Street:      "  Test Street  ",
			HouseNumber: "  10  ",
		},
	)
	if err != nil {
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
	}

	if address == nil {
		t.Fatal("expected address, got nil")
	}

	if repository.createInput == nil {
		t.Fatal("expected input to be passed to repository")
	}

	if repository.createInput.City != "Amsterdam" {
		t.Errorf(
			"expected city %q, got %q",
			"Amsterdam",
			repository.createInput.City,
		)
	}

	if repository.createInput.Street != "Test Street" {
		t.Errorf(
			"expected street %q, got %q",
			"Test Street",
			repository.createInput.Street,
		)
	}

	if repository.createInput.HouseNumber != "10" {
		t.Errorf(
			"expected house number %q, got %q",
			"10",
			repository.createInput.HouseNumber,
		)
	}
}

func TestService_Create_Validation(t *testing.T) {
	tests := []struct {
		name  string
		input *CreateAddress
	}{
		{
			name:  "nil input",
			input: nil,
		},
		{
			name: "invalid user id",
			input: &CreateAddress{
				UserID:      0,
				City:        "Amsterdam",
				Street:      "Test Street",
				HouseNumber: "10",
			},
		},
		{
			name: "empty city",
			input: &CreateAddress{
				UserID:      10,
				City:        "   ",
				Street:      "Test Street",
				HouseNumber: "10",
			},
		},
		{
			name: "empty street",
			input: &CreateAddress{
				UserID:      10,
				City:        "Amsterdam",
				Street:      "   ",
				HouseNumber: "10",
			},
		},
		{
			name: "empty house number",
			input: &CreateAddress{
				UserID:      10,
				City:        "Amsterdam",
				Street:      "Test Street",
				HouseNumber: "   ",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := &fakeRepository{}
			service := NewService(repository)

			address, err := service.Create(
				context.Background(),
				tt.input,
			)

			if address != nil {
				t.Fatalf(
					"expected nil address, got %+v",
					address,
				)
			}

			if !errors.Is(
				err,
				ErrAddressValidation,
			) {
				t.Fatalf(
					"expected ErrAddressValidation, got %v",
					err,
				)
			}

			if repository.createInput != nil {
				t.Fatal(
					"repository must not be called for invalid input",
				)
			}
		})
	}
}

func TestService_Create_RepositoryError(t *testing.T) {
	repositoryErr := errors.New(
		"database unavailable",
	)

	repository := &fakeRepository{
		createErr: repositoryErr,
	}

	service := NewService(repository)

	address, err := service.Create(
		context.Background(),
		&CreateAddress{
			UserID:      10,
			City:        "Amsterdam",
			Street:      "Test Street",
			HouseNumber: "10",
		},
	)

	if address != nil {
		t.Fatalf(
			"expected nil address, got %+v",
			address,
		)
	}

	if !errors.Is(err, repositoryErr) {
		t.Fatalf(
			"expected repository error, got %v",
			err,
		)
	}
}

func TestService_GetByID(t *testing.T) {
	repository := &fakeRepository{
		getByIDAddress: &Address{
			ID:          5,
			UserID:      10,
			City:        "Amsterdam",
			Street:      "Test Street",
			HouseNumber: "10",
		},
	}

	service := NewService(repository)

	address, err := service.GetByID(
		context.Background(),
		5,
		10,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if address == nil {
		t.Fatal("expected address, got nil")
	}

	if address.ID != 5 {
		t.Errorf(
			"expected address ID %d, got %d",
			5,
			address.ID,
		)
	}

	if repository.getByIDID != 5 {
		t.Errorf(
			"expected repository address ID %d, got %d",
			5,
			repository.getByIDID,
		)
	}

	if repository.getByIDUserID != 10 {
		t.Errorf(
			"expected repository user ID %d, got %d",
			10,
			repository.getByIDUserID,
		)
	}
}

func TestService_GetByID_Validation(t *testing.T) {
	tests := []struct {
		name      string
		addressID int64
		userID    int64
	}{
		{
			name:      "invalid address id",
			addressID: 0,
			userID:    10,
		},
		{
			name:      "invalid user id",
			addressID: 5,
			userID:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := &fakeRepository{}
			service := NewService(repository)

			address, err := service.GetByID(
				context.Background(),
				tt.addressID,
				tt.userID,
			)

			if address != nil {
				t.Fatalf(
					"expected nil address, got %+v",
					address,
				)
			}

			if !errors.Is(err, ErrAddressValidation) {
				t.Fatalf(
					"expected ErrAddressValidation, got %v",
					err,
				)
			}
		})
	}
}

func TestService_GetByID_NotFound(t *testing.T) {
	repository := &fakeRepository{
		getByIDErr: pgx.ErrNoRows,
	}

	service := NewService(repository)

	address, err := service.GetByID(
		context.Background(),
		999,
		10,
	)

	if address != nil {
		t.Fatalf(
			"expected nil address, got %+v",
			address,
		)
	}

	if !errors.Is(err, ErrAddressNotFound) {
		t.Fatalf(
			"expected ErrAddressNotFound, got %v",
			err,
		)
	}
}

func TestService_GetByID_RepositoryError(t *testing.T) {
	repositoryErr := errors.New("database unavailable")

	repository := &fakeRepository{
		getByIDErr: repositoryErr,
	}

	service := NewService(repository)

	address, err := service.GetByID(
		context.Background(),
		5,
		10,
	)

	if address != nil {
		t.Fatalf(
			"expected nil address, got %+v",
			address,
		)
	}

	if !errors.Is(err, repositoryErr) {
		t.Fatalf(
			"expected repository error, got %v",
			err,
		)
	}
}

func TestService_ListByUser(t *testing.T) {
	repository := &fakeRepository{
		listByUserAddresses: []Address{
			{
				ID:          1,
				UserID:      10,
				City:        "Amsterdam",
				Street:      "First Street",
				HouseNumber: "1",
			},
			{
				ID:          2,
				UserID:      10,
				City:        "Amsterdam",
				Street:      "Second Street",
				HouseNumber: "2",
			},
		},
	}

	service := NewService(repository)

	addresses, err := service.ListByUser(
		context.Background(),
		10,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(addresses) != 2 {
		t.Fatalf(
			"expected %d addresses, got %d",
			2,
			len(addresses),
		)
	}

	if repository.listByUserID != 10 {
		t.Errorf(
			"expected repository user ID %d, got %d",
			10,
			repository.listByUserID,
		)
	}
}

func TestService_ListByUser_InvalidUserID(t *testing.T) {
	repository := &fakeRepository{}
	service := NewService(repository)

	addresses, err := service.ListByUser(
		context.Background(),
		0,
	)

	if addresses != nil {
		t.Fatalf(
			"expected nil addresses, got %+v",
			addresses,
		)
	}

	if !errors.Is(err, ErrAddressValidation) {
		t.Fatalf(
			"expected ErrAddressValidation, got %v",
			err,
		)
	}
}

func TestService_ListByUser_RepositoryError(t *testing.T) {
	repositoryErr := errors.New("database unavailable")

	repository := &fakeRepository{
		listByUserErr: repositoryErr,
	}

	service := NewService(repository)

	addresses, err := service.ListByUser(
		context.Background(),
		10,
	)

	if addresses != nil {
		t.Fatalf(
			"expected nil addresses, got %+v",
			addresses,
		)
	}

	if !errors.Is(err, repositoryErr) {
		t.Fatalf(
			"expected repository error, got %v",
			err,
		)
	}
}

func TestService_Delete(t *testing.T) {
	repository := &fakeRepository{}
	service := NewService(repository)

	err := service.Delete(
		context.Background(),
		5,
		10,
	)
	if err != nil {
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
	}

	if repository.deleteAddressID != 5 {
		t.Errorf(
			"expected address ID %d, got %d",
			5,
			repository.deleteAddressID,
		)
	}

	if repository.deleteUserID != 10 {
		t.Errorf(
			"expected user ID %d, got %d",
			10,
			repository.deleteUserID,
		)
	}
}

func TestService_Delete_Validation(t *testing.T) {
	tests := []struct {
		name      string
		addressID int64
		userID    int64
	}{
		{
			name:      "invalid address id",
			addressID: 0,
			userID:    10,
		},
		{
			name:      "invalid user id",
			addressID: 5,
			userID:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := &fakeRepository{}
			service := NewService(repository)

			err := service.Delete(
				context.Background(),
				tt.addressID,
				tt.userID,
			)

			if !errors.Is(
				err,
				ErrAddressValidation,
			) {
				t.Fatalf(
					"expected ErrAddressValidation, got %v",
					err,
				)
			}

			if repository.deleteAddressID != 0 {
				t.Fatal(
					"repository must not be called for invalid input",
				)
			}
		})
	}
}

func TestService_Delete_NotFound(t *testing.T) {
	repository := &fakeRepository{
		deleteErr: ErrAddressNotFound,
	}

	service := NewService(repository)

	err := service.Delete(
		context.Background(),
		999,
		10,
	)

	if !errors.Is(err, ErrAddressNotFound) {
		t.Fatalf(
			"expected ErrAddressNotFound, got %v",
			err,
		)
	}
}

func TestService_Delete_RepositoryError(t *testing.T) {
	repositoryErr := errors.New(
		"database unavailable",
	)

	repository := &fakeRepository{
		deleteErr: repositoryErr,
	}

	service := NewService(repository)

	err := service.Delete(
		context.Background(),
		5,
		10,
	)

	if !errors.Is(err, repositoryErr) {
		t.Fatalf(
			"expected repository error, got %v",
			err,
		)
	}
}

func TestService_Update(t *testing.T) {
	repository := &fakeRepository{
		updateAddress: &Address{
			ID:          5,
			UserID:      10,
			City:        "Rotterdam",
			Street:      "New Street",
			HouseNumber: "20",
		},
	}

	service := NewService(repository)

	city := "  Rotterdam  "
	street := "  New Street  "
	houseNumber := "  20  "

	address, err := service.Update(
		context.Background(),
		5,
		10,
		&UpdateAddress{
			City:        &city,
			Street:      &street,
			HouseNumber: &houseNumber,
		},
	)
	if err != nil {
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
	}

	if address == nil {
		t.Fatal("expected address, got nil")
	}

	if repository.updateAddressID != 5 {
		t.Errorf(
			"expected address ID %d, got %d",
			5,
			repository.updateAddressID,
		)
	}

	if repository.updateUserID != 10 {
		t.Errorf(
			"expected user ID %d, got %d",
			10,
			repository.updateUserID,
		)
	}

	if repository.updateInput == nil {
		t.Fatal("expected update input")
	}

	if *repository.updateInput.City != "Rotterdam" {
		t.Errorf(
			"expected city %q, got %q",
			"Rotterdam",
			*repository.updateInput.City,
		)
	}

	if *repository.updateInput.Street != "New Street" {
		t.Errorf(
			"expected street %q, got %q",
			"New Street",
			*repository.updateInput.Street,
		)
	}

	if *repository.updateInput.HouseNumber != "20" {
		t.Errorf(
			"expected house number %q, got %q",
			"20",
			*repository.updateInput.HouseNumber,
		)
	}
}

func TestService_Update_Validation(t *testing.T) {
	empty := "   "

	tests := []struct {
		name      string
		addressID int64
		userID    int64
		input     *UpdateAddress
	}{
		{
			name:      "invalid address id",
			addressID: 0,
			userID:    10,
			input:     &UpdateAddress{},
		},
		{
			name:      "invalid user id",
			addressID: 5,
			userID:    0,
			input:     &UpdateAddress{},
		},
		{
			name:      "nil input",
			addressID: 5,
			userID:    10,
			input:     nil,
		},
		{
			name:      "empty city",
			addressID: 5,
			userID:    10,
			input: &UpdateAddress{
				City: &empty,
			},
		},
		{
			name:      "empty street",
			addressID: 5,
			userID:    10,
			input: &UpdateAddress{
				Street: &empty,
			},
		},
		{
			name:      "empty house number",
			addressID: 5,
			userID:    10,
			input: &UpdateAddress{
				HouseNumber: &empty,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := &fakeRepository{}
			service := NewService(repository)

			address, err := service.Update(
				context.Background(),
				tt.addressID,
				tt.userID,
				tt.input,
			)

			if address != nil {
				t.Fatalf(
					"expected nil address, got %+v",
					address,
				)
			}

			if !errors.Is(
				err,
				ErrAddressValidation,
			) {
				t.Fatalf(
					"expected ErrAddressValidation, got %v",
					err,
				)
			}

			if repository.updateInput != nil {
				t.Fatal(
					"repository must not be called for invalid input",
				)
			}
		})
	}
}

func TestService_Update_NotFound(t *testing.T) {
	repository := &fakeRepository{
		updateErr: pgx.ErrNoRows,
	}

	service := NewService(repository)

	city := "Rotterdam"

	address, err := service.Update(
		context.Background(),
		999,
		10,
		&UpdateAddress{
			City: &city,
		},
	)

	if address != nil {
		t.Fatalf(
			"expected nil address, got %+v",
			address,
		)
	}

	if !errors.Is(err, ErrAddressNotFound) {
		t.Fatalf(
			"expected ErrAddressNotFound, got %v",
			err,
		)
	}
}

func TestService_Update_RepositoryError(t *testing.T) {
	repositoryErr := errors.New(
		"database unavailable",
	)

	repository := &fakeRepository{
		updateErr: repositoryErr,
	}

	service := NewService(repository)

	city := "Rotterdam"

	address, err := service.Update(
		context.Background(),
		5,
		10,
		&UpdateAddress{
			City: &city,
		},
	)

	if address != nil {
		t.Fatalf(
			"expected nil address, got %+v",
			address,
		)
	}

	if !errors.Is(err, repositoryErr) {
		t.Fatalf(
			"expected repository error, got %v",
			err,
		)
	}
}
