<template>
	<div class="box entry">
		<article class="media">
			<div class="media-content">
				<div class="content">
					<div class="level icon-row is-mobile">
						<div class="level-left">
							<font-awesome-icon
								icon="user"
								class="mr-2 level-item"
								fixed-width
							/>
							<span class="level-item stay-in-container">
								<strong>{{ name }}</strong>
							</span>
						</div>
					</div>
					<span v-if="!anonymous">
						<div class="level icon-row is-mobile">
							<div class="level-left">
								<font-awesome-icon
									icon="at"
									class="mr-2 level-item"
									fixed-width
								/>
								<span class="level-item stay-in-container">{{
									entry.email
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
									entry.description
								}}</span>
							</div>
						</div>
						<div class="level icon-row is-mobile">
							<div class="level-left">
								<font-awesome-icon
									:icon="queue.config.virtual ? 'link' : 'map-marker'"
									class="mr-2 level-item"
									fixed-width
								/>
								<p class="level-item link-in-container" v-html="location"></p>
							</div>
						</div>
					</span>
					<div class="level icon-row is-mobile">
						<div class="level-left">
							<font-awesome-icon
								icon="clock"
								class="mr-2 level-item"
								fixed-width
							/>
							<b-tooltip :label="entry.tooltipTimestamp">
								<span class="level-item stay-in-container">{{
									humanizedTimestamp
								}}</span>
							</b-tooltip>
						</div>
					</div>
					<div class="level icon-row is-mobile" v-if="entry.priority !== 0">
						<div class="level-left">
							<font-awesome-icon
								icon="sort-numeric-up"
								class="mr-2 level-item"
								fixed-width
								v-if="entry.priority > 0"
							/>
							<font-awesome-icon
								icon="sort-numeric-down"
								class="mr-2 level-item"
								fixed-width
								v-else
							/>
							<span class="level-item stay-in-container"
								>Priority:
								{{ (entry.priority > 0 ? '+' : '') + entry.priority }}</span
							>
						</div>
					</div>
					<div class="level icon-row is-mobile" v-if="stack">
						<div class="level-left">
							<font-awesome-icon
								icon="times"
								class="mr-2 level-item"
								fixed-width
							/>
							<span class="level-item stay-in-container">{{
								entry.removedBy
							}}</span>
						</div>
					</div>
					<div v-if="!anonymous">
						<br />
						<div class="buttons is-fullwidth">
							<template v-if="!stack">
								<button
									class="button is-success"
									:class="{ 'is-loading': removeRequestRunning }"
									v-on:click="removeEntry"
									v-if="admin"
								>
									<span class="icon"
										><font-awesome-icon icon="hands-helping"
									/></span>
									<span>Help</span>
								</button>
								<button
									class="button is-danger"
									:class="{ 'is-loading': removeRequestRunning }"
									v-on:click="removeEntry"
									v-else
								>
									<span class="icon"><font-awesome-icon icon="times"/></span>
									<span>Cancel</span>
								</button>
							</template>
							<template v-if="!entry.pinned && admin">
								<button
									class="button is-primary"
									:class="{ 'is-loading': pinEntryRequestRunning }"
									v-on:click="pinEntry"
								>
									<span class="icon"
										><font-awesome-icon icon="thumbtack"
									/></span>
									<span>Pin</span>
								</button>
							</template>
							<template v-if="admin">
								<button class="button is-warning" @click="messageUser">
									<span class="icon"><font-awesome-icon icon="envelope"/></span>
									<span>Message</span>
								</button>
							</template>
						</div>
					</div>
				</div>
			</div>
			<figure class="media-right" style="padding: 0">
				<div class="is-pulled-right" v-if="admin">
					<b-tooltip
						:label="
							'This student is currently ' +
								(entry.online ? 'online' : 'offline') +
								'.'
						"
						:class="{
							'is-success': entry.online,
							'is-danger': !entry.online,
						}"
						position="is-left"
					>
						<font-awesome-icon
							:style="{
								color: entry.online
									? 'hsl(141, 53%, 53%)'
									: 'hsl(348, 100%, 61%)',
							}"
							class="is-size-6"
							icon="circle"
							fixed-width
						/>
					</b-tooltip>
				</div>
				<div class="is-pulled-right" v-if="entry.pinned">
					<b-tooltip
						label="This student is pinned to the top of the queue."
						position="is-left"
					>
						<font-awesome-icon
							class="is-size-1 is-size-6-touch"
							icon="thumbtack"
							fixed-width
						/>
					</b-tooltip>
				</div>
				<div class="is-pulled-right" v-if="stack && !entry.helped">
					<b-tooltip
						label="This student wasn't able to be helped."
						position="is-left"
					>
						<font-awesome-icon
							icon="frown-open"
							class="is-size-1 is-size-6-touch"
							fixed-width
						/>
					</b-tooltip>
				</div>
			</figure>
		</article>
	</div>
</template>

<script lang="ts">
import { Component, Vue, Prop } from 'vue-property-decorator';
import moment, { Moment } from 'moment-timezone';
import linkifyStr from 'linkifyjs/string';
import OrderedQueue from '@/types/OrderedQueue';
import { QueueEntry } from '@/types/QueueEntry';
import ErrorDialog from '@/util/ErrorDialog';

import { library } from '@fortawesome/fontawesome-svg-core';
import {
	faUser,
	faAt,
	faQuestion,
	faLink,
	faClock,
	faSortNumericUp,
	faSortNumericDown,
	faTimes,
	faThumbtack,
	faEnvelope,
	faHandsHelping,
	faFrownOpen,
	faMapMarker,
	faCircle,
} from '@fortawesome/free-solid-svg-icons';

library.add(
	faUser,
	faAt,
	faQuestion,
	faLink,
	faClock,
	faSortNumericUp,
	faSortNumericDown,
	faTimes,
	faThumbtack,
	faEnvelope,
	faHandsHelping,
	faFrownOpen,
	faMapMarker,
	faCircle
);

@Component
export default class QueueEntryDisplay extends Vue {
	@Prop({ required: true }) entry!: QueueEntry;
	@Prop({ required: true }) stack!: boolean;
	@Prop({ required: true }) queue!: OrderedQueue;
	@Prop({ required: true }) admin!: boolean;
	@Prop({ required: true }) time!: Moment;

	get anonymous() {
		return !(
			this.admin ||
			(this.$root.$data.userInfo.email !== undefined &&
				this.entry.email === this.$root.$data.userInfo.email)
		);
	}

	get name() {
		return this.anonymous ? 'Anonymous Student' : this.entry.name;
	}

	get location() {
		return linkifyStr(this.entry.location || '', {
			defaultProtocol: 'https',
		});
	}

	get humanizedTimestamp() {
		// HACK: fix time update lag issues in the beginning
		// by saying the time is 5 seconds ahead of what it really is.
		// Since we only display "a few seconds ago" this shouldn't have
		// any noticeable impact.
		return this.entry.humanizedTimestamp(this.time.clone().add(5, 'second'));
	}

	removeRequestRunning = false;
	removeEntry() {
		this.queue.personallyRemovedEntries.add(this.entry.id);
		this.removeRequestRunning = true;
		fetch(
			process.env.BASE_URL +
				`api/queues/${this.queue.id}/entries/${this.entry.id}`,
			{
				method: 'DELETE',
			}
		).then((res) => {
			this.removeRequestRunning = false;
			if (res.status !== 204) {
				return ErrorDialog(res);
			}
		});
	}

	pinEntryRequestRunning = false;
	pinEntry() {
		this.pinEntryRequestRunning = true;
		fetch(
			process.env.BASE_URL +
				`api/queues/${this.queue.id}/entries/${this.entry.id}/pin`,
			{
				method: 'POST',
			}
		).then((res) => {
			this.pinEntryRequestRunning = false;
			if (res.status !== 204) {
				return ErrorDialog(res);
			}

			this.$buefy.toast.open({
				duration: 5000,
				message: `Pinned ${this.entry.email}!`,
				type: 'is-success',
			});
		});
	}

	messageUser() {
		this.$buefy.dialog.prompt({
			message: `Send message to ${this.entry.email}:`,
			inputAttrs: {
				placeholder: 'Your meeting is empty, please come back!',
			},
			trapFocus: true,
			onConfirm: (message) => {
				fetch(process.env.BASE_URL + `api/queues/${this.queue.id}/messages`, {
					method: 'POST',
					body: JSON.stringify({
						receiver: this.entry.email,
						content: message,
					}),
				}).then((res) => {
					if (res.status !== 201) {
						return ErrorDialog(res);
					}

					this.$buefy.toast.open({
						duration: 5000,
						message: `Sent message to ${this.entry.email}`,
						type: 'is-success',
					});
				});
			},
		});
	}
}
</script>

<style scoped>
.entry {
	overflow-x: hidden;
}

.icon-row {
	margin-bottom: 0px;
}

.level-left {
	flex-shrink: 1;
}

.stay-in-container {
	flex-shrink: 1;
	overflow-wrap: break-word;
	word-break: break-word;
	hyphens: auto;
}

.link-in-container {
	flex-shrink: 1;
	word-break: break-all;
	overflow-wrap: anywhere;
	hyphens: auto;
	display: inline-block;
}
</style>
