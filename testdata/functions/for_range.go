package for_range

func CountOnly(n int32) int32 {
	count := 0
	for range n {
		count++
	}
	return count
}

func SumRange(n int32) int32 {
	sum := 0
	for i := range n {
		sum += i
	}
	return sum
}

func SumRangeLiteral() int32 {
	sum := 0
	for i := range 6 {
		sum += i
	}
	return sum
}

func SumRangeI64(n int64) int64 {
	var sum int64
	for i := range n {
		sum += i
	}
	return sum
}

func SumRangeU32(n uint32) uint32 {
	var sum uint32
	for i := range n {
		sum += i
	}
	return sum
}

func RangeBreakContinue(n int32) int32 {
	sum := 0
	for i := range n {
		if i%2 != 0 {
			continue
		}
		if i == 6 {
			break
		}
		sum += i
	}
	return sum
}

func RangeMutateTargetAndKey(n int32) int32 {
	sum := 0
	for i := range n {
		sum += i
		n = 0
		i = 100
	}
	return sum
}

func RangeAssignExisting(n int32) int32 {
	last := -1
	for last = range n {
	}
	return last
}

func RangeBlankAssign(n int32) int32 {
	count := 0
	for _ = range n {
		count += 2
	}
	return count
}

func RangeSequentialAndNested(rows, cols int32) int32 {
	total := 0
	for i := range rows {
		for j := range cols {
			total += i*10 + j
		}
	}
	for i := range rows {
		total += i
	}
	return total
}

func RangeShadowSelf(i int32) int32 {
	sum := 0
	for i := range i {
		sum += i
	}
	return sum
}
