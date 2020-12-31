<template>
	<div class="columns">
		<div class="column is-two-thirds">
			<div class="box" style="position: relative">
				<transition name="fade" mode="out-in">
					<appointments-display
						class="appointments-display"
						:queue="queue"
						:time="time"
						:appointments="appointments"
						:selectedAppointment="selectedAppointment"
						:admin="true"
						@selected="appointmentSelected"
						v-if="loaded"
						key="display"
					/>
					<b-skeleton height="10em" v-else key="loading"></b-skeleton>
				</transition>
				<button
					class="button is-white edit-schedule-button"
					@click="editSchedule"
				>
					<font-awesome-icon icon="cog" />
				</button>
			</div>
		</div>
		<div class="column is-one-third">
			<div class="box" style="height: 100%">
				<p class="title is-5" v-if="selectedAppointment === null">
					Select an appointment!
				</p>
				<div v-else>
					<div v-if="selectedAppointment.filled">
						<p class="title is-5">
							Appointment at
							{{ selectedAppointment.scheduledTime.format('LT') }}
						</p>
						<div class="block">
							<p v-if="selectedAppointment.studentEmail === undefined">
								No student yet!
							</p>
							<span v-else>
								<div class="level icon-row is-mobile">
									<div class="level-left">
										<font-awesome-icon
											icon="signature"
											class="mr-2 level-item"
											fixed-width
										/>
										<strong class="level-item stay-in-container">{{
											selectedAppointment.name
										}}</strong>
									</div>
								</div>
								<div class="level icon-row is-mobile">
									<div class="level-left">
										<font-awesome-icon
											icon="at"
											class="mr-2 level-item"
											fixed-width
										/>
										<span class="level-item stay-in-container">{{
											selectedAppointment.studentEmail
										}}</span>
									</div>
								</div>
								<div class="level icon-row is-mobile">
									<div class="level-left">
										<font-awesome-icon
											icon="question"
											class="mr-2 level-item"
											fixed-width
										/>
										<span class="level-item stay-in-container">{{
											selectedAppointment.description
										}}</span>
									</div>
								</div>
								<div class="level icon-row is-mobile">
									<div class="level-left">
										<font-awesome-icon
											icon="link"
											class="mr-2 level-item"
											fixed-width
										/>
										<p
											class="level-item link-in-container"
											v-html="selectedAppointmentLocation"
										></p>
									</div></div
							></span>
							<div class="level icon-row is-mobile">
								<div class="level-left">
									<font-awesome-icon
										icon="chalkboard-teacher"
										class="mr-2 level-item"
										fixed-width
									/>
									<strong class="level-item stay-in-container">{{
										selectedAppointment.staffEmail || '(unclaimed)'
									}}</strong>
								</div>
							</div>
						</div>
						<button
							class="button is-success"
							v-if="selectedAppointment.staffEmail === undefined"
							@click="claimTimeslot"
						>
							<span class="icon"><font-awesome-icon icon="hand-paper"/></span>
							<span>Claim</span>
						</button>
						<button
							class="button is-danger"
							v-else-if="
								selectedAppointment.staffEmail === $root.$data.userInfo.email
							"
							@click="unclaimAppointment"
						>
							<span class="icon"
								><font-awesome-icon icon="calendar-times"
							/></span>
							<span>Unclaim</span>
						</button>
					</div>
					<div v-else>
						<p class="title is-5">
							Empty slot at {{ selectedAppointment.scheduledTime.format('LT') }}
						</p>
						<button class="button is-success" @click="claimTimeslot">
							<span class="icon"><font-awesome-icon icon="hand-paper"/></span>
							<span>Claim</span>
						</button>
					</div>
				</div>
			</div>
		</div>
	</div>
</template>

<script lang="ts">
import Vue from 'vue';
import { Moment } from 'moment';
import { Component, Prop, Watch } from 'vue-property-decorator';
import linkifyStr from 'linkifyjs/string';
import { AppointmentsQueue } from '@/types/AppointmentsQueue';
import AppointmentsDisplay from '@/components/appointments/AppointmentsDisplay.vue';
import AppointmentsSchedule from '@/components/appointments/AppointmentsSchedule.vue';
import { Appointment, AppointmentSlot } from '@/types/Appointment';
import ErrorDialog from '@/util/ErrorDialog';

import { library } from '@fortawesome/fontawesome-svg-core';
import {
	faSignature,
	faAt,
	faQuestion,
	faLink,
	faChalkboardTeacher,
	faHandPaper,
	faCalendarTimes,
	faCog,
} from '@fortawesome/free-solid-svg-icons';

library.add(
	faSignature,
	faAt,
	faQuestion,
	faLink,
	faChalkboardTeacher,
	faHandPaper,
	faCalendarTimes,
	faCog
);

@Component({
	components: { AppointmentsDisplay },
})
export default class AppointmentsAdminSelector extends Vue {
	name = '';
	description = '';
	location = '';

	@Prop({ required: true }) queue!: AppointmentsQueue;
	@Prop({ required: true }) time!: Moment;
	@Prop({ required: true }) loaded!: boolean;

	get appointments() {
		if (this.queue.schedule === undefined) {
			return undefined;
		}

		return this.queue.schedule.appointmentSlots;
	}

	appointmentSelected(timeslot: number | null, index: number | null) {
		this.selectedTimeslot = timeslot;
		this.selectedIndex = index;
	}

	selectedTimeslot: number | null = null;
	selectedIndex: number | null = null;

	get selectedAppointment(): AppointmentSlot | null {
		if (
			this.selectedTimeslot === null ||
			this.selectedIndex === null ||
			this.queue.schedule === undefined
		) {
			return null;
		}

		return this.queue.schedule.timeslots[this.selectedTimeslot]!.slots[
			this.selectedIndex
		]!;
	}

	get selectedAppointmentLocation(): string {
		if (
			this.selectedAppointment === null ||
			!this.selectedAppointment.filled ||
			(this.selectedAppointment as Appointment).location === undefined
		) {
			return '';
		}

		return linkifyStr((this.selectedAppointment as Appointment).location!, {
			defaultProtocol: 'https',
		});
	}

	@Watch('time')
	onTimeUpdated() {
		if (
			this.selectedAppointment !== null &&
			this.selectedAppointment.scheduledTime
				.clone()
				.add(this.selectedAppointment.duration, 'minutes') < this.time
		) {
			this.appointmentSelected(null, null);
		}
	}

	claimTimeslot() {
		fetch(
			process.env.BASE_URL +
				`api/queues/${this.queue.id}/appointments/${this.queue.schedule?.day}/claims/${this.selectedTimeslot}`,
			{
				method: 'PUT',
			}
		).then((res) => {
			if (res.status !== 201) {
				return ErrorDialog(res);
			}
		});
	}

	unclaimAppointment() {
		fetch(
			process.env.BASE_URL +
				`api/queues/${this.queue.id}/appointments/claims/${
					(this.selectedAppointment as Appointment).id
				}`,
			{
				method: 'DELETE',
			}
		).then((res) => {
			if (res.status !== 201) {
				return ErrorDialog(res);
			}
		});
	}

	editSchedule() {
		fetch(
			process.env.BASE_URL + `api/queues/${this.queue.id}/appointments/schedule`
		)
			.then((res) => res.json())
			.then((schedule: [{ [index: string]: any }]) => {
				schedule.sort(
					(a: { [index: string]: any }, b: { [index: string]: any }) =>
						a['day'] - b['day']
				);
				const schedules = schedule.map((s: { [index: string]: any }) =>
					s['schedule'].split('').map((i: string) => parseInt(i))
				);
				const durations = schedule.map(
					(s: { [index: string]: any }) => s['duration']
				);
				this.$buefy.modal.open({
					parent: this,
					component: AppointmentsSchedule,
					props: { defaultSchedules: schedules, defaultDurations: durations },
					events: {
						saved: (day: number, duration: number, schedule: number[]) => {
							const scheduleStr = schedule
								.map((slot: number) => slot.toString())
								.join('');
							fetch(
								process.env.BASE_URL +
									`api/queues/${this.queue.id}/appointments/schedule/${day}`,
								{
									method: 'PUT',
									body: JSON.stringify({
										duration: duration,
										padding: 2,
										schedule: scheduleStr,
									}),
								}
							).then((res) => {
								if (res.status !== 204) {
									return ErrorDialog(res);
								}
							});
						},
					},
					hasModalCard: true,
					trapFocus: true,
				});
			});
	}
}
</script>

<style scoped>
.appointments-display {
	overflow-x: scroll;
	white-space: nowrap;
}

.icon-row {
	margin-bottom: 0px;
}

.edit-schedule-button {
	position: absolute;
	top: 10px;
	right: 10px;
}
</style>