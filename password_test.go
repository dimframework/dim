package dim

import (
	"testing"

	"golang.org/x/crypto/bcrypt"
)

// TestBcryptCostDefault menjaga bawaan produksi. Diperiksa terhadap konstanta,
// bukan terhadap BcryptCost saat test berjalan, agar tetap bermakna bila suatu
// saat paket ini menurunkan cost lewat TestMain seperti yang didokumentasikan.
func TestBcryptCostDefault(t *testing.T) {
	if defaultBcryptCost != 12 {
		t.Errorf("defaultBcryptCost = %d, want 12", defaultBcryptCost)
	}
}

// TestHashPasswordUsesBcryptCost memastikan BcryptCost benar-benar dipakai saat
// hashing — inilah yang membuat penurunan cost di TestMain berefek.
//
// Menulis global yang dibaca HashPassword. Aman karena tidak ada satu pun test
// di paket ini yang memakai t.Parallel(); bila itu berubah, test ini perlu
// dipindahkan ke jalur yang tidak menyentuh global.
func TestHashPasswordUsesBcryptCost(t *testing.T) {
	original := BcryptCost
	t.Cleanup(func() { BcryptCost = original })

	BcryptCost = bcrypt.MinCost

	hash, err := HashPassword("TestPassword123!")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	cost, err := bcrypt.Cost([]byte(hash))
	if err != nil {
		t.Fatalf("bcrypt.Cost() error = %v", err)
	}
	if cost != bcrypt.MinCost {
		t.Errorf("cost hash = %d, want %d", cost, bcrypt.MinCost)
	}

	// Hash dengan cost rendah tetap harus dapat diverifikasi.
	if err := VerifyPassword(hash, "TestPassword123!"); err != nil {
		t.Errorf("VerifyPassword() gagal untuk hash cost %d: %v", bcrypt.MinCost, err)
	}
}

// TestHashPasswordRejectsOutOfRangeCost menjaga lubang yang terbuka begitu
// BcryptCost dapat ditulis: bcrypt menukar cost di bawah MinCost dengan
// DefaultCost-nya sendiri (10) tanpa galat. Sebuah field config yang tidak
// terisi — `dim.BcryptCost = cfg.BcryptCost` yang bernilai 0 — akan melemahkan
// setiap kata sandi tanpa gejala apa pun. Cost di luar rentang harus gagal.
func TestHashPasswordRejectsOutOfRangeCost(t *testing.T) {
	original := BcryptCost
	t.Cleanup(func() { BcryptCost = original })

	for _, cost := range []int{0, -1, bcrypt.MinCost - 1, bcrypt.MaxCost + 1} {
		BcryptCost = cost

		hash, err := HashPassword("TestPassword123!")
		if err == nil {
			actual, _ := bcrypt.Cost([]byte(hash))
			t.Errorf("BcryptCost = %d: HashPassword() err = nil (hash cost %d), want error", cost, actual)
		}
	}
}

func TestHashPassword(t *testing.T) {
	password := "TestPassword123!"

	hash, err := HashPassword(password)
	if err != nil {
		t.Errorf("HashPassword() error = %v", err)
	}

	if hash == "" {
		t.Errorf("hash is empty")
	}

	if hash == password {
		t.Errorf("hash should not equal password")
	}
}

func TestVerifyPassword(t *testing.T) {
	password := "TestPassword123!"

	hash, _ := HashPassword(password)

	err := VerifyPassword(hash, password)
	if err != nil {
		t.Errorf("VerifyPassword() should succeed for correct password")
	}
}

func TestVerifyPasswordIncorrect(t *testing.T) {
	password := "TestPassword123!"

	hash, _ := HashPassword(password)

	err := VerifyPassword(hash, "WrongPassword")
	if err == nil {
		t.Errorf("VerifyPassword() should fail for incorrect password")
	}
}

func TestPasswordValidatorMinLength(t *testing.T) {
	validator := NewPasswordValidator().SetMinLength(10)

	err := validator.Validate("Short1!")
	if err == nil {
		t.Errorf("Validate() should fail for password shorter than minimum length")
	}
}

func TestPasswordValidatorRequireUppercase(t *testing.T) {
	validator := NewPasswordValidator().
		SetMinLength(8).
		RequireLowercase(false).
		RequireDigit(false).
		RequireSpecial(false)

	err := validator.Validate("lowercasepassword")
	if err == nil {
		t.Errorf("Validate() should fail without uppercase")
	}
}

func TestPasswordValidatorRequireLowercase(t *testing.T) {
	validator := NewPasswordValidator().
		SetMinLength(8).
		RequireUppercase(false).
		RequireDigit(false).
		RequireSpecial(false)

	err := validator.Validate("UPPERCASEPASSWORD")
	if err == nil {
		t.Errorf("Validate() should fail without lowercase")
	}
}

func TestPasswordValidatorRequireDigit(t *testing.T) {
	validator := NewPasswordValidator().
		SetMinLength(8).
		RequireUppercase(false).
		RequireLowercase(false).
		RequireSpecial(false)

	err := validator.Validate("NoDigitPassword")
	if err == nil {
		t.Errorf("Validate() should fail without digit")
	}
}

func TestPasswordValidatorRequireSpecial(t *testing.T) {
	validator := NewPasswordValidator().
		SetMinLength(8).
		RequireUppercase(false).
		RequireLowercase(false).
		RequireDigit(false)

	err := validator.Validate("NoSpecialPassword123")
	if err == nil {
		t.Errorf("Validate() should fail without special character")
	}
}

func TestPasswordValidatorValidPassword(t *testing.T) {
	validator := NewPasswordValidator()

	err := validator.Validate("ValidPass123!")
	if err != nil {
		t.Errorf("Validate() should succeed for valid password: %v", err)
	}
}

func TestPasswordValidatorDisableRequirements(t *testing.T) {
	validator := NewPasswordValidator().
		SetMinLength(0).
		RequireUppercase(false).
		RequireLowercase(false).
		RequireDigit(false).
		RequireSpecial(false)

	err := validator.Validate("a")
	if err != nil {
		t.Errorf("Validate() should succeed when all requirements disabled: %v", err)
	}
}

func TestValidatePasswordStrength(t *testing.T) {
	tests := []struct {
		password string
		wantErr  bool
	}{
		{"ValidPass123!", false},
		{"weak", true},
		{"NoDigit!", true},
		{"nouppercase123!", true},
		{"NOLOWERCASE123!", true},
		{"NoSpecial123", true},
	}

	for _, tt := range tests {
		err := ValidatePasswordStrength(tt.password)
		if (err != nil) != tt.wantErr {
			t.Errorf("ValidatePasswordStrength(%s) error = %v, wantErr %v", tt.password, err, tt.wantErr)
		}
	}
}

func TestHashPasswordDifferentHashes(t *testing.T) {
	password := "SamePassword123!"

	hash1, _ := HashPassword(password)
	hash2, _ := HashPassword(password)

	// Hashes should be different due to salt
	if hash1 == hash2 {
		t.Errorf("hashes for same password should be different")
	}

	// But both should verify correctly
	if err := VerifyPassword(hash1, password); err != nil {
		t.Errorf("VerifyPassword() failed for hash1")
	}

	if err := VerifyPassword(hash2, password); err != nil {
		t.Errorf("VerifyPassword() failed for hash2")
	}
}
