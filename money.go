package money

import (
	"errors"
	"math"
	"math/big"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

var ZeroMoney, _ = primitive.ParseDecimal128("0")

func Add(m, m1 primitive.Decimal128) (primitive.Decimal128, error) {
	c, e, err := m.BigInt()
	if err != nil {
		return ZeroMoney, err
	}

	c1, e1, err := m1.BigInt()
	if err != nil {
		return ZeroMoney, err
	}
	if e < -323 || e > 308 || e1 < -323 || e1 > 308 {
		return ZeroMoney, errors.New("invalid")
	}
	if e == e1 {
		bgRes := big.NewInt(c.Int64() + c1.Int64())
		val, ok := primitive.ParseDecimal128FromBigInt(bgRes, e)
		if !ok {
			return ZeroMoney, nil
		}
		return val, nil
	} else if e < e1 {
		factor := math.Pow10(e1 - e)
		c1 = c1.Mul(c1, big.NewInt(int64(factor)))
		bgRes := c1.Add(c1, c)
		val, _ := primitive.ParseDecimal128FromBigInt(bgRes, e)
		return val, nil
	}
	factor := math.Pow10(e - e1)
	c = c.Mul(c, big.NewInt(int64(factor)))
	bgRes := c.Add(c, c1)
	val, _ := primitive.ParseDecimal128FromBigInt(bgRes, e1)
	return val, nil
}

func Sub(m, m1 primitive.Decimal128) (primitive.Decimal128, error) {
	c, e, err := m.BigInt()
	if err != nil {
		return ZeroMoney, err
	}

	c1, e1, err := m1.BigInt()
	if err != nil {
		return ZeroMoney, err
	}
	if e < -323 || e > 308 || e1 < -323 || e1 > 308 {
		return ZeroMoney, errors.New("invalid")
	}
	c1.Mul(c1, big.NewInt(-1))
	if e == e1 {
		bgRes := big.NewInt(c.Int64() + c1.Int64())
		val, _ := primitive.ParseDecimal128FromBigInt(bgRes, e)
		return val, nil
	} else if e < e1 {
		factor := math.Pow10(e1 - e)
		c1 = c1.Mul(c1, big.NewInt(int64(factor)))
		bgRes := c1.Add(c1, c)
		val, _ := primitive.ParseDecimal128FromBigInt(bgRes, e)
		return val, nil
	}
	factor := math.Pow10(e - e1)
	c = c.Mul(c, big.NewInt(int64(factor)))
	bgRes := c.Add(c, c1)
	val, _ := primitive.ParseDecimal128FromBigInt(bgRes, e1)
	return val, nil
}

func Mul(m, m1 primitive.Decimal128) (primitive.Decimal128, error) {
	c, e, err := m.BigInt()
	if err != nil {
		return ZeroMoney, err
	}

	c1, e1, err := m1.BigInt()
	if err != nil {
		return ZeroMoney, err
	}

	cMul := c.Mul(c, c1)
	eSum := e + e1
	res, _ := primitive.ParseDecimal128FromBigInt(cMul, eSum)
	return res, nil
}

func SystemRound(roundTo primitive.Decimal128, roundDir string, val primitive.Decimal128) primitive.Decimal128 {
	if roundTo == ZeroMoney {
		return val
	}
	c, e, _ := val.BigInt()
	rtc, rte, _ := roundTo.BigInt()
	if e > rte {
		c = c.Mul(c, big.NewInt(int64(math.Pow10(e-rte))))
		e = rte
	}
	rtc = rtc.Mul(rtc, big.NewInt(int64(math.Pow10(rte-e))))
	rem := new(big.Int).Set(c).Mod(c, rtc)
	roundedVal := c
	if rem.Cmp(big.NewInt(0)) != 0 {
		roundedVal = roundedVal.Sub(roundedVal, rem)
		if roundDir == "up" {
			roundedVal.Add(roundedVal, rtc)
		}
	}
	roundedVal.Float64()
	roundedRes, _ := primitive.ParseDecimal128FromBigInt(roundedVal, e)
	return roundedRes
}
