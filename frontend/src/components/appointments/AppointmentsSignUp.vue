<template>
	<div>
		<div class="field is-horizontal">
			<div class="field-label">
				<label class="label">Description</label>
			</div>
			<div class="field-body">
				<div class="field">
					<div class="control has-icons-left">
						<input
							class="input"
							v-model="description"
							type="text"
							placeholder="Help us help you—please be descriptive!"
						/>
						<span class="icon is-small is-left">
							<font-awesome-icon icon="question" />
						</span>
					</div>
				</div>
			</div>
		</div>
		<div class="field is-horizontal">
			<div class="field-label">
				<label class="label">Location/Meeting Link</label>
			</div>
			<div class="field-body">
				<div class="field">
					<div class="control has-icons-left">
						<input class="input" v-model="location" type="text" />
						<span class="icon is-small is-left">
							<font-awesome-icon icon="map-marker" />
						</span>
					</div>
				</div>
			</div>
		</div>
		<div class="field is-horizontal">
			<div class="field-label">
				<label class="label">Appointments</label>
			</div>
			<div class="field-body" style="min-width: 0;">
				<div class="field" style="width: 100%;">
					<div class="box">
						<transition name="fade" mode="out-in">
							<appointments-display
								class="appointments-display"
								:queue="queue"
								:time="time"
								:appointments="studentAppointments"
								:selectedAppointment="selectedAppointment"
								:admin="false"
								@selected="appointmentSelected"
								v-if="loaded"
								key="display"
							/>
							<b-skeleton height="10em" v-else key="loading"></b-skeleton>
						</transition>
					</div>
				</div>
			</div>
		</div>
		<div class="field is-horizontal">
			<div class="field-label"></div>
			<div class="field-body">
				<div class="field">
					<div class="control level-left">
						<button
							class="button is-success level-item"
							v-if="selectedAppointment === null"
							disabled
						>
							<span class="icon"><font-awesome-icon icon="user-clock"/></span>
							<span>Select a time slot!</span>
						</button>
						<button
							class="button is-success level-item"
							:class="{ 'is-loading': signUpRequstRunning }"
							:disabled="!canSignUp"
							v-else-if="myAppointment === null"
							@click="signUp"
						>
							<span class="icon"
								><font-awesome-icon icon="calendar-check"
							/></span>
							<span
								>Schedule appointment at
								{{ selectedAppointment.scheduledTime.format('LT') }}</span
							>
						</button>
						<button
							class="button is-warning level-item"
							:class="{ 'is-loading': updateAppointmentRequestRunning }"
							v-else-if="myAppointmentModified"
							@click="updateAppointment"
						>
							<span class="icon"><font-awesome-icon icon="edit"/></span>
							<span>Update Appointment</span>
						</button>
						<button class="button is-success level-item" disabled="true" v-else>
							<span class="icon"><font-awesome-icon icon="check"/></span>
							<span
								>Scheduled for
								{{ myAppointment.scheduledTime.format('LT') }}</span
							>
						</button>
						<button
							class="button is-danger level-item"
							:class="{ 'is-loading': cancelAppointmentRequestRunning }"
							v-if="myAppointment !== null"
							@click="cancelAppointment"
						>
							<span class="icon"
								><font-awesome-icon icon="calendar-times"
							/></span>
							<span>Cancel Appointment</span>
						</button>
						<p class="level-item" v-if="!$root.$data.loggedIn">
							Log in to sign up!
						</p>
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
import { AppointmentsQueue } from '@/types/AppointmentsQueue';
import AppointmentsDisplay from '@/components/appointments/AppointmentsDisplay.vue';
import { Appointment, AppointmentSlot } from '@/types/Appointment';
import ErrorDialog from '@/util/ErrorDialog';
import EscapeHTML from '@/util/Sanitization';

import { library } from '@fortawesome/fontawesome-svg-core';
import {
	faUser,
	faQuestion,
	faLink,
	faCalendarCheck,
	faCalendarTimes,
	faCheck,
	faUserClock,
	faEdit,
	faMapMarker,
} from '@fortawesome/free-solid-svg-icons';

library.add(
	faUser,
	faQuestion,
	faLink,
	faCalendarCheck,
	faCalendarTimes,
	faCheck,
	faUserClock,
	faEdit,
	faMapMarker
);

@Component({
	components: { AppointmentsDisplay },
})
export default class AppointmentsSignUp extends Vue {
	description = '';
	location = '';

	@Prop({ required: true }) queue!: AppointmentsQueue;
	@Prop({ required: true }) time!: Moment;
	@Prop({ required: true }) loaded!: boolean;

	get studentAppointments() {
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

	@Watch('time')
	onTimeUpdated() {
		if (
			this.selectedAppointment !== null &&
			this.queue.schedule !== undefined &&
			!(
				this.myAppointment !== null &&
				this.myAppointment.timeslot === this.selectedAppointment.timeslot
			) &&
			this.queue.schedule.timeslots[this.selectedAppointment.timeslot].past(
				this.time
			)
		) {
			this.appointmentSelected(null, null);
		}
	}

	@Watch('myAppointment', { immediate: true })
	myAppointmentUpdated(
		newAppointment: Appointment | null,
		oldAppointment: Appointment | null
	) {
		if (newAppointment !== oldAppointment && newAppointment !== null) {
			this.description = newAppointment.description || '';
			this.location = newAppointment.location || '';
			this.appointmentSelected(
				newAppointment.timeslot,
				this.queue.schedule?.timeslots[newAppointment.timeslot]?.slots.indexOf(
					newAppointment
				)!
			);
		}
	}

	get myAppointment(): Appointment | null {
		if (
			this.$root.$data.userInfo.email === undefined ||
			this.queue.schedule === undefined
		) {
			return null;
		}

		for (const timeslot of Object.values(this.queue.schedule.timeslots)) {
			for (const slot of timeslot.slots) {
				if (
					slot.filled &&
					(slot as Appointment).studentEmail ===
						this.$root.$data.userInfo.email &&
					slot.scheduledTime
						.clone()
						.add(this.queue.schedule.duration, 'minutes')
						.diff(this.time) > 0
				) {
					return slot as Appointment;
				}
			}
		}

		return null;
	}

	get myAppointmentModified() {
		const a = this.myAppointment;
		return (
			a !== null &&
			(a.description !== this.description ||
				a.location !== this.location ||
				(this.selectedAppointment !== null &&
					a.timeslot !== this.selectedAppointment.timeslot))
		);
	}

	get canSignUp() {
		return (
			this.$root.$data.loggedIn &&
			this.description !== undefined &&
			this.location !== undefined &&
			this.description.trim() !== '' &&
			this.location.trim() !== ''
		);
	}

	signUp() {
		if (this.queue.config?.confirmSignupMessage !== undefined) {
			return this.$buefy.dialog.confirm({
				title: 'Sign Up',
				message: EscapeHTML(this.queue.config!.confirmSignupMessage),
				type: 'is-warning',
				hasIcon: true,
				onConfirm: this.signUpRequest,
			});
		}

		this.signUpRequest();
	}

	signUpRequstRunning = false;
	signUpRequest() {
		this.signUpRequstRunning = true;
		fetch(
			process.env.BASE_URL +
				`api/queues/${this.queue.id}/appointments/${this.queue.schedule?.day}/${this.selectedAppointment?.timeslot}`,
			{
				method: 'POST',
				body: JSON.stringify({
					description: this.description,
					location: this.location,
				}),
			}
		).then((res) => {
			this.signUpRequstRunning = false;
			if (res.status !== 201) {
				return ErrorDialog(res);
			}

			const startTime: Moment = this.selectedAppointment?.scheduledTime!;
			const endTime: Moment = startTime
				.clone()
				.add(this.selectedAppointment?.duration, 'minutes');
			const link = new URL('http://www.google.com/calendar/event');
			link.searchParams.append('action', 'TEMPLATE');
			link.searchParams.append(
				'dates',
				startTime.toISOString().replaceAll(/[-:\\.]/g, '') +
					'/' +
					endTime.toISOString().replaceAll(/[-:\\.]/g, '')
			);
			link.searchParams.append(
				'text',
				this.queue.course.shortName + ' Office Hours Appointment'
			);
			link.searchParams.append('location', this.location);

			this.$buefy.dialog.alert({
				title: 'Appointment Scheduled',
				message: `Success, ${EscapeHTML(
					this.$root.$data.userInfo.first_name
				)}—your appointment has been scheduled. Make sure to be ready at ${this.selectedAppointment?.scheduledTime.format(
					'LT'
				)}! <a href="${link}" target="_blank">Add this appointment to your Google calendar.</a>`,
				type: 'is-success',
				hasIcon: true,
			});
		});
	}

	updateAppointmentRequestRunning = false;
	updateAppointment() {
		if (
			this.myAppointment !== null &&
			this.selectedAppointment?.timeslot !== this.myAppointment.timeslot &&
			this.myAppointment.scheduledTime.diff(this.time) < 0
		) {
			return this.$buefy.dialog.alert({
				title: 'Slow Down!',
				message: `An appointment's time can't be changed while it's happening!`,
				type: 'is-danger',
				hasIcon: true,
			});
		}

		this.updateAppointmentRequestRunning = true;
		fetch(
			process.env.BASE_URL +
				`api/queues/${this.queue.id}/appointments/${this.myAppointment?.id}`,
			{
				method: 'PUT',
				body: JSON.stringify({
					description: this.description,
					location: this.location,
					timeslot: this.selectedAppointment?.timeslot,
				}),
			}
		).then((res) => {
			this.updateAppointmentRequestRunning = false;
			if (res.status !== 201) {
				return ErrorDialog(res);
			}

			this.$buefy.dialog.alert({
				title: 'Appointment Updated',
				message: `Your appointment has been updated. Make sure to be ready at ${EscapeHTML(
					this.selectedAppointment?.scheduledTime.format('LT')!
				)}!`,
				type: 'is-success',
				hasIcon: true,
			});
		});
	}

	cancelAppointmentRequestRunning = false;
	cancelAppointment() {
		this.$buefy.dialog.confirm({
			title: 'Cancel Appointment',
			message: `Are you sure you want to cancel your ${EscapeHTML(
				this.myAppointment?.scheduledTime.format('LT')!
			)} appointment?`,
			type: 'is-danger',
			hasIcon: true,
			confirmText: 'Cancel Appointment',
			cancelText: 'Close',
			onConfirm: () => {
				this.cancelAppointmentRequestRunning = true;
				fetch(
					process.env.BASE_URL +
						`api/queues/${this.queue.id}/appointments/${this.myAppointment?.id}`,
					{
						method: 'DELETE',
					}
				).then((res) => {
					this.cancelAppointmentRequestRunning = false;
					if (res.status !== 204) {
						return ErrorDialog(res);
					}

					this.$buefy.dialog.alert({
						title: 'Appointment Canceled',
						message: `Your appointment has been canceled.`,
						type: 'is-success',
						hasIcon: true,
					});
				});
			},
		});
	}
}
</script>

<style scoped>
.appointments-display {
	overflow-x: scroll;
	white-space: nowrap;
}
</style>
