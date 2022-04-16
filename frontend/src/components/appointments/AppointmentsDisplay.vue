<template>
	<div :id="'student-appointments-display-' + id">
		<div
			class="appointment-slots-group"
			v-for="(group, i) in consecutiveTimeslots"
			:key="'group-' + i"
		>
			<appointments-column
				class="appointment-column"
				v-for="(pair, j) in group"
				:key="'group-' + i + '-timeslot-' + j"
				:id="'student-appointment-timeslot-' + pair[0].timeslot + '-' + id"
				:admin="admin"
				:appointments="pair[1]"
				:timeslot="pair[0]"
				:index="j"
				:time="time"
				:selected="
					selectedAppointment !== null &&
						selectedAppointment.timeslot === pair[0].timeslot
				"
				@selected="
					(indexInTimeslot) =>
						$emit('selected', pair[0].timeslot, indexInTimeslot)
				"
			/>
		</div>
	</div>
</template>

<script lang="ts">
import { Component, Prop, Watch, Vue } from 'vue-property-decorator';
import moment, { Moment } from 'moment-timezone';
import {
	AppointmentsQueue,
	AppointmentsTimeslot,
} from '@/types/AppointmentsQueue';
import AppointmentsColumn from '@/components/appointments/AppointmentsColumn.vue';
import { Appointment, AppointmentSlot } from '@/types/Appointment';

@Component({
	components: {
		AppointmentsColumn,
	},
})
export default class AppointmentsDisplay extends Vue {
	@Prop({ required: true }) queue!: AppointmentsQueue;
	@Prop({ required: true }) time!: Moment;
	@Prop({ required: true }) admin!: boolean;

	@Prop({ required: true }) appointments!: {
		[index: number]: AppointmentSlot[];
	};

	@Prop({ required: true }) selectedAppointment!: AppointmentSlot | null;

	id: string;

	constructor() {
		super();
		this.id = Math.random()
			.toString(36)
			.substr(2);
	}

	get firstOpenTimeslot(): number | undefined {
		if (this.queue.schedule === undefined) {
			return undefined;
		}

		const s = Object.entries(this.queue.schedule.timeslots).find(
			(t: [string, AppointmentsTimeslot]) =>
				!t[1].past(
					this.time
						.clone()
						.subtract(this.admin ? this.queue.schedule?.duration : 0, 'minutes')
				)
		);

		return s !== undefined ? parseInt(s[0]) : undefined;
	}

	mounted() {
		if (this.firstOpenTimeslot !== undefined) {
			this.scrollToTimeslot(this.firstOpenTimeslot, false);
		}
	}

	@Watch('firstOpenTimeslot')
	onTimeUpdated(newSlot: number | undefined, oldSlot: number | undefined) {
		if (newSlot !== undefined && newSlot !== oldSlot) {
			this.scrollToTimeslot(newSlot, true);
		}
	}

	scrollToTimeslot(timeslot: number, smooth: boolean) {
		// Find DOM element corresponding to column of timeslot
		const timeslotElement = document.getElementById(
			'student-appointment-timeslot-' + timeslot + '-' + this.id
		);
		const appointmentsDisplayElement = document.getElementById(
			'student-appointments-display-' + this.id
		);
		if (timeslotElement === null || appointmentsDisplayElement === null) {
			return;
		}

		// Scroll appointments display to area of first open timeslot
		const delta =
			timeslotElement.getBoundingClientRect().left -
			appointmentsDisplayElement.getBoundingClientRect().left;
		appointmentsDisplayElement.scroll({
			left: appointmentsDisplayElement.scrollLeft + delta - 50,
			behavior: smooth ? 'smooth' : 'auto',
		});
	}

	get consecutiveTimeslots(): [AppointmentsTimeslot, AppointmentSlot[]][][] {
		const groups: [AppointmentsTimeslot, AppointmentSlot[]][][] = [];
		const slots = Object.keys(this.appointments).map((n) => parseInt(n));

		if (slots.length === 0 || this.queue.schedule === undefined) {
			return [];
		}

		let lastSeen: number = slots[0];
		let running: [AppointmentsTimeslot, AppointmentSlot[]][] = [
			[this.queue.schedule.timeslots[lastSeen], this.appointments[lastSeen]],
		];
		for (let i = 1; i < slots.length; i++) {
			if (slots[i] !== lastSeen + 1) {
				groups.push(running);
				running = [];
			}

			running.push([
				this.queue.schedule.timeslots[slots[i]],
				this.appointments[slots[i]],
			]);
			lastSeen = slots[i];
		}

		groups.push(running);
		return groups;
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

	/* Leave some space for the scroll bar */
	margin-bottom: 1em;
}
</style>
