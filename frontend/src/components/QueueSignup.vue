<template>
	<div>
		<div class="field">
			<label class="label">Name</label>
			<div class="control has-icons-left">
				<input class="input" v-model="name" type="text" placeholder="Nice to meet you!" />
				<span class="icon is-small is-left">
					<font-awesome-icon icon="user" />
				</span>
			</div>
		</div>
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
		<div class="field">
			<label class="label">Meeting Link</label>
			<div class="control has-icons-left">
				<input class="input" v-model="location" type="text" />
				<span class="icon is-small is-left">
					<font-awesome-icon icon="link" />
				</span>
			</div>
		</div>
		<div class="field">
			<div class="control level-left">
				<button
					class="button is-success level-item"
					:disabled="!canSignUp"
					@click="signUp"
					v-if="myEntry === undefined"
				>Sign Up</button>
				<button
					class="button is-warning level-item"
					@click="updateRequest"
					v-else-if="myEntryModified"
				>Update Request</button>
				<button class="button is-success level-item" disabled="true" v-else>Signed up</button>
				<p class="level-item" v-if="!$root.$data.loggedIn">Log in to sign up!</p>
			</div>
		</div>
	</div>
</template>

<script lang="ts">
import Vue from 'vue';
import { Moment } from 'moment';
import { Component, Prop, Watch } from 'vue-property-decorator';
import OrderedQueue from '../types/OrderedQueue';
import { QueueEntry } from '../types/QueueEntry';
import ErrorDialog from '../util/ErrorDialog';

import { library } from '@fortawesome/fontawesome-svg-core';
import { faUser, faQuestion, faLink } from '@fortawesome/free-solid-svg-icons';

library.add(faUser, faQuestion, faLink);

@Component
export default class QueueSignup extends Vue {
	name = '';
	description = '';
	location = '';

	@Prop({ required: true }) queue!: OrderedQueue;
	@Prop({ required: true }) time!: Moment;

	@Watch('myEntry')
	myEntryUpdated(
		newEntry: QueueEntry | undefined,
		oldEntry: QueueEntry | undefined
	) {
		if (newEntry !== undefined) {
			this.name = newEntry.name;
			this.description = newEntry.description;
			this.location = newEntry.location;
		}
	}

	get canSignUp() {
		return (
			this.$root.$data.loggedIn &&
			this.queue.open(this.time) &&
			this.name !== undefined &&
			this.description !== undefined &&
			this.location !== undefined &&
			this.name.trim() !== '' &&
			this.description.trim() !== '' &&
			this.location.trim() !== '' &&
			this.myEntry === undefined
		);
	}

	get myEntry() {
		if (this.$root.$data.userInfo.email === undefined) {
			return undefined;
		}

		return this.queue.entries.find(
			(e) => e.email === this.$root.$data.userInfo.email
		);
	}

	get myEntryModified() {
		const e = this.myEntry;
		return (
			e !== undefined &&
			(e.name !== this.name ||
				e.description !== this.description ||
				e.location !== this.location)
		);
	}

	signUp() {
		fetch(process.env.BASE_URL + `api/queues/${this.queue.id}/entries`, {
			method: 'POST',
			body: JSON.stringify({
				name: this.name,
				description: this.description,
				location: this.location,
			}),
		}).then((res) => {
			if (res.status !== 201) {
				return ErrorDialog(res);
			}

			this.$buefy.toast.open({
				duration: 5000,
				message: `You're on the queue!`,
				type: 'is-success',
			});
		});
	}

	updateRequest() {
		if (this.myEntry !== undefined) {
			fetch(
				process.env.BASE_URL +
					`api/queues/${this.queue.id}/entries/${this.myEntry.id}`,
				{
					method: 'PUT',
					body: JSON.stringify({
						name: this.name,
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
