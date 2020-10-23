<template>
	<div>
		<div
			class="appointment-slots-group"
			v-for="group in queue.schedule.consecutiveTimeslots"
			:key="'group-'+group[0]"
		>
			<appointments-column
				class="appointment-column"
				v-for="(slot, index) in group"
				:key="slot"
				:slot="slot"
				:index="index"
				:time="time"
				:myAppointment="myAppointment !== undefined && myAppointment.timeslot === slot.timeslot"
				:selected="selectedTimeslot === slot.timeslot"
				@selected="$emit('selected', slot.timeslot, slot.time)"
			/>
		</div>
	</div>
</template>

<script lang="ts">
import { Component, Prop, Watch, Vue } from 'vue-property-decorator';
import moment, { Moment } from 'moment-timezone';
import { AppointmentsQueue } from '@/types/AppointmentsQueue';
import AppointmentsColumn from '@/components/appointments/student-display/AppointmentsColumn.vue';
import Appointment from '@/types/Appointment';

@Component({
	components: {
		AppointmentsColumn,
	},
})
export default class AppointmentsStudentDisplay extends Vue {
	@Prop({ required: true }) queue!: AppointmentsQueue;
	@Prop({ required: true }) time!: Moment;

	@Prop({ required: true }) selectedTimeslot!: number | null;
	@Prop({ required: true }) selectedTime!: Moment | null;

	@Prop({ required: true }) myAppointment!: Appointment | undefined;

	@Watch('time')
	onTimeUpdated() {
		if (
			this.selectedTimeslot !== null &&
			this.queue.schedule !== undefined &&
			!(
				this.myAppointment !== undefined &&
				this.myAppointment.timeslot === this.selectedTimeslot
			) &&
			this.queue.schedule.timeslots[this.selectedTimeslot].past(this.time)
		) {
			this.$emit('selected', null, null);
		}
	}
}
</script>

<style scoped>
.appointment-slots-group {
	display: inline-block;
	padding-right: 1em;
}

.appointment-column {
	display: inline-block;
	margin-right: 2px;
	vertical-align: top;
	width: 1.5em;

	/* Leave some space for the scroll bar */
	margin-bottom: 1em;
}
</style>
