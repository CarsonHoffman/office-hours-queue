<template>
	<div>
		<div class="field">
			<label class="label">Description</label>
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
		<div
			class="field"
			v-if="queue.config === null || queue.config.enableLocationField"
		>
			<label class="label" v-if="queue.config === null"
				><b-skeleton width="7em"
			/></label>
			<label class="label" v-else-if="!queue.config.virtual">Location</label>
			<label class="label" v-else>Meeting Link</label>
			<div class="control has-icons-left">
				<input class="input" v-model="location" type="text" />
				<span class="icon is-small is-left">
					<b-skeleton
						position="is-centered"
						width="1em"
						v-if="queue.config === null"
					/>
					<font-awesome-icon
						icon="map-marker"
						v-else-if="!queue.config.virtual"
					/>
					<font-awesome-icon icon="link" v-else />
				</span>
			</div>
		</div>
		<div class="field">
			<div class="control level-left">
				<button
					class="button is-success level-item"
					:disabled="!canSignUp"
					@click="signUp"
					v-if="myEntry === null"
				>
					<span class="icon"><font-awesome-icon icon="user-plus"/></span>
					<span>Sign Up</span>
				</button>
				<button
					class="button is-warning level-item"
					@click="updateRequest"
					v-else-if="myEntryModified"
				>
					<span class="icon"><font-awesome-icon icon="edit"/></span>
					<span>Update Request</span>
				</button>
				<button class="button is-success level-item" disabled="true" v-else>
					<span class="icon"><font-awesome-icon icon="check"/></span>
					<span>On queue at position #{{ this.myEntryIndex + 1 }}</span>
				</button>
				<p class="level-item" v-if="!$root.$data.loggedIn">
					Log in to sign up!
				</p>
			</div>
		</div>
	</div>
</template>

<script lang="ts">
import Vue from 'vue';
import { Moment } from 'moment';
import { Component, Prop, Watch } from 'vue-property-decorator';
import OrderedQueue from '@/types/OrderedQueue';
import { QueueEntry } from '@/types/QueueEntry';
import ErrorDialog from '@/util/ErrorDialog';
import EscapeHTML from '@/util/Sanitization';

import { library } from '@fortawesome/fontawesome-svg-core';
import {
	faUser,
	faQuestion,
	faLink,
	faUserPlus,
	faCheck,
	faEdit,
	faMapMarker,
} from '@fortawesome/free-solid-svg-icons';

library.add(
	faUser,
	faQuestion,
	faLink,
	faUserPlus,
	faCheck,
	faEdit,
	faMapMarker
);

@Component
export default class QueueSignup extends Vue {
	description = '';
	location = '';

	@Prop({ required: true }) queue!: OrderedQueue;
	@Prop({ required: true }) time!: Moment;

	@Watch('myEntry')
	myEntryUpdated(newEntry: QueueEntry | null) {
		if (newEntry !== null) {
			this.description = newEntry.description || '';
			this.location = newEntry.location || '';
		}
	}

	get canSignUp(): boolean {
		// Do not change the order of the expressions in this boolean
		// expression. Because myEntry is a computed property, it seems
		// that it has to be calculated at least once in order to be
		// reactive, which means that putting it at the end of the
		// expression means it isn't calculated until all of the previous
		// parts of the expression are true, which is only calculated when
		// deemed necessary based on reactivity. Thus, if one of the previous
		// parts of the expression return false on the first calculation,
		// we aren't reactive on myEntry until one of the previous
		// parts had a reactive update. This took way too long to figure out :(
		return (
			this.myEntry === null &&
			this.$root.$data.loggedIn &&
			this.queue.isOpen(this.time) &&
			this.description.trim() !== '' &&
			(this.location.trim() !== '' || !this.queue.config?.enableLocationField)
		);
	}

	get myEntryIndex(): number {
		return this.queue.entryIndex(this.$root.$data.userInfo.email);
	}

	get myEntry(): QueueEntry | null {
		return this.queue.entry(this.$root.$data.userInfo.email);
	}

	get myEntryModified() {
		const e = this.myEntry;
		return (
			e !== null &&
			(e.description !== this.description || e.location !== this.location)
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

	signUpRequest() {
		// No, this doesn't prevent students from manually hitting the API to specify
		// a location. l33t h4x!
		const location = this.queue.config?.enableLocationField
			? this.location
			: '(disabled)';
		fetch(process.env.BASE_URL + `api/queues/${this.queue.id}/entries`, {
			method: 'POST',
			body: JSON.stringify({
				description: this.description,
				location,
			}),
		}).then((res) => {
			if (res.status !== 201) {
				return ErrorDialog(res);
			}

			this.$buefy.toast.open({
				duration: 5000,
				message: `You're on the queue, ${EscapeHTML(
					this.$root.$data.userInfo.first_name
				)}!`,
				type: 'is-success',
			});
		});
	}

	updateRequest() {
		if (this.myEntry !== null) {
			fetch(
				process.env.BASE_URL +
					`api/queues/${this.queue.id}/entries/${this.myEntry.id}`,
				{
					method: 'PUT',
					body: JSON.stringify({
						description: this.description,
						location: this.location,
					}),
				}
			).then((res) => {
				if (res.status !== 204) {
					return ErrorDialog(res);
				}

				this.$buefy.toast.open({
					duration: 5000,
					message: 'Your request has been updated!',
					type: 'is-success',
				});
			});
		}
	}
}
</script>
