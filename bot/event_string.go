// Code generated by "stringer -type=Event events.go"; DO NOT EDIT.

package bot

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[IgnoredUser-0]
	_ = x[BotDirectMessage-1]
	_ = x[AdminCheckPassed-2]
	_ = x[AdminCheckFailed-3]
	_ = x[MultipleMatchesNoAction-4]
	_ = x[AuthNoRunMisconfigured-5]
	_ = x[AuthNoRunPlugNotAvailable-6]
	_ = x[AuthRanSuccess-7]
	_ = x[AuthRanFail-8]
	_ = x[AuthRanMechanismFailed-9]
	_ = x[AuthRanFailNormal-10]
	_ = x[AuthRanFailOther-11]
	_ = x[AuthNoRunNotFound-12]
	_ = x[ElevNoRunMisconfigured-13]
	_ = x[ElevNoRunNotAvailable-14]
	_ = x[ElevRanSuccess-15]
	_ = x[ElevRanFail-16]
	_ = x[ElevRanMechanismFailed-17]
	_ = x[ElevRanFailNormal-18]
	_ = x[ElevRanFailOther-19]
	_ = x[ElevNoRunNotFound-20]
	_ = x[CommandTaskRan-21]
	_ = x[AmbientTaskRan-22]
	_ = x[CatchAllsRan-23]
	_ = x[CatchAllTaskRan-24]
	_ = x[TriggeredTaskRan-25]
	_ = x[SpawnedTaskRan-26]
	_ = x[ScheduledTaskRan-27]
	_ = x[JobTaskRan-28]
	_ = x[GoPluginRan-29]
	_ = x[ExternalTaskBadPath-30]
	_ = x[ExternalTaskBadInterpreter-31]
	_ = x[ExternalTaskRan-32]
	_ = x[ExternalTaskStderrOutput-33]
	_ = x[ExternalTaskErrExit-34]
}

const _Event_name = "IgnoredUserBotDirectMessageAdminCheckPassedAdminCheckFailedMultipleMatchesNoActionAuthNoRunMisconfiguredAuthNoRunPlugNotAvailableAuthRanSuccessAuthRanFailAuthRanMechanismFailedAuthRanFailNormalAuthRanFailOtherAuthNoRunNotFoundElevNoRunMisconfiguredElevNoRunNotAvailableElevRanSuccessElevRanFailElevRanMechanismFailedElevRanFailNormalElevRanFailOtherElevNoRunNotFoundCommandTaskRanAmbientTaskRanCatchAllsRanCatchAllTaskRanTriggeredTaskRanSpawnedTaskRanScheduledTaskRanJobTaskRanGoPluginRanExternalTaskBadPathExternalTaskBadInterpreterExternalTaskRanExternalTaskStderrOutputExternalTaskErrExit"

var _Event_index = [...]uint16{0, 11, 27, 43, 59, 82, 104, 129, 143, 154, 176, 193, 209, 226, 248, 269, 283, 294, 316, 333, 349, 366, 380, 394, 406, 421, 437, 451, 467, 477, 488, 507, 533, 548, 572, 591}

func (i Event) String() string {
	if i < 0 || i >= Event(len(_Event_index)-1) {
		return "Event(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _Event_name[_Event_index[i]:_Event_index[i+1]]
}
