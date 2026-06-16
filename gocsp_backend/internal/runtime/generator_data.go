package runtime

import (
	"fmt"
	"strconv"
	"strings"
)

func expandGeneratorData(
	data []any,
) ([]any, error) {

	var result []any

	for _, item := range data {

		switch v := item.(type) {

		case string:

			// диапазон вида 1..10
			if strings.Contains(v, "..") {

				parts := strings.Split(v, "..")

				if len(parts) != 2 {
					return nil,
						fmt.Errorf(
							"invalid range: %s",
							v,
						)
				}

				start, err :=
					strconv.Atoi(
						strings.TrimSpace(parts[0]),
					)

				if err != nil {
					return nil, err
				}

				end, err :=
					strconv.Atoi(
						strings.TrimSpace(parts[1]),
					)

				if err != nil {
					return nil, err
				}

				step := 1

				if start > end {
					step = -1
				}

				for i := start; ; i += step {

					result =
						append(result, i)

					if i == end {
						break
					}
				}

				continue
			}

			// обычная строка
			result = append(result, v)

		default:

			result = append(result, v)
		}
	}

	return result, nil
}