package for_loop

func SumToN(n int32) int32 {
	sum := 0
	for i := 0; i <= n; i++ {
		sum += i
	}
	return sum
}

func Factorial(n int32) int32 {
	result := 1
	for n > 1 {
		result *= n
		n--
	}
	return result
}

func SumEven(n int32) int32 {
	sum := 0
	for i := 0; i <= n; i++ {
		if i%2 != 0 {
			continue
		}
		sum += i
	}
	return sum
}

func FindBreak(target int32) int32 {
	count := 0
	for {
		if count == target {
			break
		}
		count++
	}
	return count
}

func NestedLoops(rows, cols int32) int32 {
	total := 0
	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			if c == 2 {
				break
			}
			total++
		}
	}
	return total
}

func Countdown(start int32) int32 {
	count := 0
	for i := start; i > 0; i-- {
		count++
	}
	return count
}

func CompoundPost(n int32) int32 {
	count := 0
	for i := 0; i < n; i += 2 {
		count++
	}
	return count
}
