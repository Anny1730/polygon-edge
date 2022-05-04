package ibft

import (
	"fmt"

	"github.com/0xPolygon/polygon-edge/types"
	"github.com/umbracle/fastrlp"
)

var (
	// IstanbulDigest represents a hash of "Istanbul practical byzantine fault tolerance"
	// to identify whether the block is from Istanbul consensus engine
	IstanbulDigest = types.StringToHash("0x63746963616c2062797a616e74696e65206661756c7420746f6c6572616e6365")

	// IstanbulExtraVanity represents a fixed number of extra-data bytes reserved for proposer vanity
	IstanbulExtraVanity = 32

	// IstanbulExtraSeal represents the fixed number of extra-data bytes reserved for proposer seal
	IstanbulExtraSeal = 65
)

var zeroBytes = make([]byte, 32)

// initIbftExtra initializes ExtraData in Header for IBFT Extra
func initIbftExtra(h *types.Header, validators []types.Address) error {
	return putIbftExtra(h, &IstanbulExtra{
		Validators:    validators,
		Seal:          []byte{},
		CommittedSeal: [][]byte{},
	})
}

// putIbftExtra sets the extra data field in the header to the passed in istanbul extra data
func putIbftExtra(h *types.Header, istanbulExtra *IstanbulExtra) error {
	// Pad zeros to the right up to istanbul vanity
	extra := h.ExtraData
	if len(extra) < IstanbulExtraVanity {
		extra = append(extra, zeroBytes[:IstanbulExtraVanity-len(extra)]...)
	} else {
		extra = extra[:IstanbulExtraVanity]
	}

	h.ExtraData = istanbulExtra.MarshalRLPTo(extra)

	return nil
}

// getIbftExtra returns the istanbul extra data field from the passed in header
func getIbftExtra(h *types.Header) (*IstanbulExtra, error) {
	if len(h.ExtraData) < IstanbulExtraVanity {
		return nil, fmt.Errorf("wrong extra size, expected greater than or equal to %d but actual %d", IstanbulExtraVanity, len(h.ExtraData))
	}

	data := h.ExtraData[IstanbulExtraVanity:]
	extra := &IstanbulExtra{}

	if err := extra.UnmarshalRLP(data); err != nil {
		return nil, err
	}

	return extra, nil
}

// unpackValidatorsFromIbftExtra extracts Validators from IBFT Extra in Header
func unpackValidatorsFromIbftExtra(h *types.Header) ([]types.Address, error) {
	extra, err := getIbftExtra(h)
	if err != nil {
		return nil, err
	}

	return extra.Validators, nil
}

// unpackValidatorsFromIbftExtra extracts Seal from IBFT Extra in Header
func unpackSealFromIbftExtra(h *types.Header) ([]byte, error) {
	extra, err := getIbftExtra(h)
	if err != nil {
		return nil, err
	}

	return extra.Seal, nil
}

// unpackValidatorsFromIbftExtra extracts CommittedSeal from IBFT Extra in Header
func unpackCommittedSealFromIbftExtra(h *types.Header) ([][]byte, error) {
	extra, err := getIbftExtra(h)
	if err != nil {
		return nil, err
	}

	return extra.CommittedSeal, nil
}

// packFieldIntoIbftExtra is a helper method to update fields in IBFT Extra of header
func packFieldIntoIbftExtra(h *types.Header, updateFn func(*IstanbulExtra)) error {
	extra, err := getIbftExtra(h)
	if err != nil {
		return err
	}

	updateFn(extra)

	return putIbftExtra(h, extra)
}

// packSealIntoIbftExtra set the given seal to Seal field in IBFT extra of header
func packSealIntoIbftExtra(h *types.Header, seal []byte) error {
	return packFieldIntoIbftExtra(h, func(extra *IstanbulExtra) {
		extra.Seal = seal
	})
}

// packCommittedSealIntoIbftExtra set the given committed seals to CommittedSeal field in IBFT extra of header
func packCommittedSealIntoIbftExtra(h *types.Header, seals [][]byte) error {
	return packFieldIntoIbftExtra(h, func(extra *IstanbulExtra) {
		extra.CommittedSeal = seals
	})
}

// filterIbftExtraForHash clears unnecessary fields in IBFT Extra for hash calculation
func filterIbftExtraForHash(h *types.Header) error {
	extra, err := getIbftExtra(h)
	if err != nil {
		return err
	}

	// This will effectively remove the Seal and Committed Seal fields,
	// while keeping proposer vanity and validator set
	// because extra.Validators is what we got from `h` in the first place.
	return initIbftExtra(h, extra.Validators)
}

// IstanbulExtra defines the structure of the extra field for Istanbul
type IstanbulExtra struct {
	Validators    []types.Address
	Seal          []byte
	CommittedSeal [][]byte
}

// MarshalRLPTo defines the marshal function wrapper for IstanbulExtra
func (i *IstanbulExtra) MarshalRLPTo(dst []byte) []byte {
	return types.MarshalRLPTo(i.MarshalRLPWith, dst)
}

// MarshalRLPWith defines the marshal function implementation for IstanbulExtra
func (i *IstanbulExtra) MarshalRLPWith(ar *fastrlp.Arena) *fastrlp.Value {
	vv := ar.NewArray()

	// Validators
	vals := ar.NewArray()
	for _, a := range i.Validators {
		vals.Set(ar.NewBytes(a.Bytes()))
	}

	vv.Set(vals)

	// Seal
	if len(i.Seal) == 0 {
		vv.Set(ar.NewNull())
	} else {
		vv.Set(ar.NewBytes(i.Seal))
	}

	// CommittedSeal
	if len(i.CommittedSeal) == 0 {
		vv.Set(ar.NewNullArray())
	} else {
		committed := ar.NewArray()
		for _, a := range i.CommittedSeal {
			if len(a) == 0 {
				vv.Set(ar.NewNull())
			} else {
				committed.Set(ar.NewBytes(a))
			}
		}
		vv.Set(committed)
	}

	return vv
}

// UnmarshalRLP defines the unmarshal function wrapper for IstanbulExtra
func (i *IstanbulExtra) UnmarshalRLP(input []byte) error {
	return types.UnmarshalRlp(i.UnmarshalRLPFrom, input)
}

// UnmarshalRLPFrom defines the unmarshal implementation for IstanbulExtra
func (i *IstanbulExtra) UnmarshalRLPFrom(p *fastrlp.Parser, v *fastrlp.Value) error {
	elems, err := v.GetElems()
	if err != nil {
		return err
	}

	if num := len(elems); num != 3 {
		return fmt.Errorf("not enough elements to decode istambul extra, expected 3 but found %d", num)
	}

	// Validators
	{
		vals, err := elems[0].GetElems()
		if err != nil {
			return fmt.Errorf("mismatch of RLP type for Validators, expected list but found %s", elems[0].Type())
		}
		i.Validators = make([]types.Address, len(vals))
		for indx, val := range vals {
			if err = val.GetAddr(i.Validators[indx][:]); err != nil {
				return err
			}
		}
	}

	// Seal
	{
		if i.Seal, err = elems[1].GetBytes(i.Seal); err != nil {
			return fmt.Errorf("failed to decode Seal: %w", err)
		}
	}

	// Committed
	{
		vals, err := elems[2].GetElems()
		if err != nil {
			return fmt.Errorf("mismatch of RLP type for CommittedSeal, expected list but found %s", elems[0].Type())
		}
		i.CommittedSeal = make([][]byte, len(vals))
		for indx, val := range vals {
			if i.CommittedSeal[indx], err = val.GetBytes(i.CommittedSeal[indx]); err != nil {
				return err
			}
		}
	}

	return nil
}
