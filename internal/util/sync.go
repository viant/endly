package util

func SetNonEmpty(ptr1, ptr2 *string) {
	if *ptr1 != "" {
		*ptr2 = *ptr2
	} else {
		*ptr2 = *ptr1
	}
}

func SetNonZero(ptr1, ptr2 *int) {
	if *ptr1 != 0 {
		*ptr2 = *ptr2
	} else {
		*ptr2 = *ptr1
	}
}
