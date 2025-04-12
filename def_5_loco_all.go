package scheduler

// Context to track state during combination generation
type combinationContext struct {
	maxCombinations uint8
	stopped         bool
}

type paramsGenerateAllValidCombinations struct {
	AvailableResourcesByType ResourcesPerType
	ResourcesNeededPerType   map[uint8]uint16

	UpTo uint8 // cap the number of combinations returned if number greater than zero.
}

func generateAllValidCombinations(params *paramsGenerateAllValidCombinations) []ResourcesPerType {
	// Get ordered list of resource types for consistent processing
	resourceTypes := params.AvailableResourcesByType.GetResourceTypesSorted()

	// Start recursive generation
	currentCombination := make(ResourcesPerType)

	// Use recursive backtracking to generate all combinations
	allCombinations := make([]ResourcesPerType, 0)

	ctx := &combinationContext{
		maxCombinations: params.UpTo,
	}

	generateCombinationsRecursive(
		&paramsGenerateCombinationsRecursive{
			AvailableResourcesByType: params.AvailableResourcesByType,
			ResourcesNeededPerType:   params.ResourcesNeededPerType,
			ResourceTypes:            resourceTypes,
			TypeIndex:                0,
			CurrentCombination:       currentCombination,
			Ctx:                      ctx,
		},

		&allCombinations,
	)

	return allCombinations
}

type paramsGenerateCombinationsRecursive struct {
	AvailableResourcesByType ResourcesPerType
	ResourcesNeededPerType   map[uint8]uint16
	ResourceTypes            []uint8
	TypeIndex                int
	CurrentCombination       ResourcesPerType
	Ctx                      *combinationContext
}

func generateCombinationsRecursive(params *paramsGenerateCombinationsRecursive, results *[]ResourcesPerType) {
	// Stop if we've reached the cap and cap is requested
	if params.Ctx.maxCombinations > 0 && uint8(len(*results)) >= params.Ctx.maxCombinations {
		params.Ctx.stopped = true

		return
	}

	// Base case: we've processed all resource types
	if params.TypeIndex >= len(params.ResourceTypes) {
		// Make a deep copy of the current combination
		finalCombination := make(ResourcesPerType)

		for rType, resources := range params.CurrentCombination {
			finalCombination[rType] = append([]*ResourceScheduled{}, resources...)
		}

		*results = append(*results, finalCombination)

		return
	}

	currentType := params.ResourceTypes[params.TypeIndex]
	neededQuantity := params.ResourcesNeededPerType[currentType]
	availableResources := params.AvailableResourcesByType[currentType]

	// Generate all combinations of resources that meet the quantity
	resourceCombinations := generateResourceCombinations(
		&paramsGenerateResourceCombinations{
			Resources:       availableResources,
			NeededQuantity:  neededQuantity,
			MaxCombinations: params.Ctx.maxCombinations,
		},
	)

	// For each valid combination of this resource type
	for _, resourceCombo := range resourceCombinations {
		// Check if we've hit the cap
		if params.Ctx.stopped {
			return
		}

		// Add this resource combo to the current combination
		params.CurrentCombination[currentType] = resourceCombo

		// Recurse to the next resource type
		generateCombinationsRecursive(
			&paramsGenerateCombinationsRecursive{
				AvailableResourcesByType: params.AvailableResourcesByType,
				ResourcesNeededPerType:   params.ResourcesNeededPerType,
				ResourceTypes:            params.ResourceTypes,
				TypeIndex:                params.TypeIndex + 1,
				CurrentCombination:       params.CurrentCombination,
				Ctx:                      params.Ctx,
			},

			results,
		)

		// Remove this resource type before trying the next combination
		delete(params.CurrentCombination, currentType)
	}
}

type paramsGenerateResourceCombinations struct {
	Resources       []*ResourceScheduled
	NeededQuantity  uint16
	MaxCombinations uint8
}

func generateResourceCombinations(params *paramsGenerateResourceCombinations) [][]*ResourceScheduled {
	results := make([][]*ResourceScheduled, 0)
	current := make([]*ResourceScheduled, 0)

	// Helper function to recursively build combinations
	var backtrack func(int, uint16)

	backtrack = func(start int, remainingNeeded uint16) {
		// Stop if we've reached the max combinations (when cap is requested)
		if params.MaxCombinations > 0 && uint8(len(results)) >= params.MaxCombinations {
			return
		}

		// If we've met the quantity requirement
		if remainingNeeded == 0 {
			// Make a copy of the current combination
			combination := make([]*ResourceScheduled, len(current))
			copy(combination, current)

			results = append(results, combination)

			return
		}

		// If we can't meet the requirement with remaining resources
		if start >= len(params.Resources) {
			return
		}

		// Try each remaining resource
		for i := start; i < len(params.Resources); i++ {
			resource := params.Resources[i]

			// If this resource provides enough to meet at least part of our need
			if resource.ServedQuantity <= remainingNeeded {
				// Include this resource
				current = append(current, resource)
				backtrack(i+1, remainingNeeded-resource.ServedQuantity)

				current = current[:len(current)-1] // Backtrack
			}
		}
	}

	backtrack(0, params.NeededQuantity)

	return results
}

func (loc *Loco) GetAllSchedulingOptions(params *ParamsCanRun) (OptionsSchedule, error) {
	intervalsSought := params.TimeInterval.BreakDown(params.TaskRun.EstimatedDuration)
	resourcesNeededPerType := params.TaskRun.GetNeededResourcesPerType()
	neededTypes := params.TaskRun.GetNeededResourceTypes()

	result := make([]*OptionSchedule, 0)

	for _, interval := range intervalsSought {
		// For each interval, first collect all available resources by type
		availableResourcesByType := make(ResourcesPerType)

		// Check if we have enough resources of each type available
		sufficientResources := true
		for _, neededType := range neededTypes {
			resourcesNeededPerCurrentType := resourcesNeededPerType[neededType]
			availableResourcesByType[neededType] = make([]*ResourceScheduled, 0)

			for _, resource := range loc.Resources[neededType] {
				if isAvailable := resource.IsAvailableIn(&interval); isAvailable {
					availableResourcesByType[neededType] = append(
						availableResourcesByType[neededType],
						resource,
					)
				}
			}

			// Check if we have enough resources of this type
			var totalAvailable uint16
			for _, res := range availableResourcesByType[neededType] {
				totalAvailable += res.ServedQuantity
			}

			if totalAvailable < resourcesNeededPerCurrentType {
				sufficientResources = false

				break
			}
		}

		if !sufficientResources {
			continue // Skip this interval
		}

		// Generate all valid combinations for this interval
		options := generateAllValidCombinations(
			&paramsGenerateAllValidCombinations{
				AvailableResourcesByType: availableResourcesByType,
				ResourcesNeededPerType:   resourcesNeededPerType,

				UpTo: params.PossibilitiesUpTo,
			},
		)

		// Create an OptionSchedule for each valid combination
		for _, resourceCombination := range options {
			result = append(
				result,
				&OptionSchedule{
					WhenCanStart: interval.TimeStart,
					Resources:    resourceCombination,
				},
			)
		}
	}

	return result,
		nil
}
