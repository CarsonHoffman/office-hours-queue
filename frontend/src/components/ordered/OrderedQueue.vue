<template>
	<div class="columns">
		<div class="column is-6">
			<div class="entries">
				<h1 class="title">Queue</h1>
				<transition name="fade" mode="out-in">
					<div v-if="loaded" key="queue-column-loaded">
						<transition name="fade" mode="out-in">
							<transition-group
								v-if="queue.entries.length > 0"
								name="entries-group"
								tag="div"
							>
								<div
									v-for="entry in queue.entries"
									:key="entry.id"
									class="block entries-group-item"
								>
									<queue-entry-display
										:entry="entry"
										:stack="false"
										:queue="queue"
										:admin="admin"
										:time="time"
									/>
								</div>
							</transition-group>
							<div class="hero is-primary" v-else>
								<div class="hero-body">
									<span v-if="!open">
										<font-awesome-icon
											icon="store-alt-slash"
											size="10x"
											class="block"
										/>
										<h1 class="title block">The queue is closed.</h1>
										<h2 class="subtitle block">
											See you next time{{
												$root.$data.loggedIn
													? ', ' + $root.$data.userInfo.first_name
													: ''
											}}!
										</h2>
									</span>
									<span v-else-if="admin">
										<font-awesome-icon
											icon="grin-hearts"
											size="10x"
											class="block"
										/>
										<h1 class="title block">The queue is empty.</h1>
										<h2 class="subtitle block">
											Good job, {{ $root.$data.userInfo.first_name }}!
										</h2>
									</span>
									<span v-else>
										<font-awesome-icon
											icon="heart-broken"
											size="10x"
											class="block"
										/>
										<h1 class="title block">The queue is empty.</h1>
										<h2 class="subtitle block">
											We're lonely over here{{
												$root.$data.loggedIn
													? ', ' + $root.$data.userInfo.first_name
													: ''
											}}!
										</h2>
									</span>
								</div>
							</div>
						</transition>
					</div>
					<div v-else>
						<div v-for="i in 10" :key="i" class="block">
							<div class="box">
								<article class="media">
									<div class="media-content">
										<div class="content">
											<b-skeleton></b-skeleton>
											<b-skeleton></b-skeleton>
										</div>
									</div>
								</article>
							</div>
						</div>
					</div>
				</transition>
			</div>
		</div>
		<div class="column is-5 is-offset-1">
			<div class="entries">
				<div class="level level-is-shrinkable" v-if="loaded">
					<div class="level-left level-is-shrinkable">
						<p class="level-item level-is-shrinkable">
							<font-awesome-icon class="mr-1" icon="user-graduate" fixed-size />
							<strong>{{ queue.entries.length }}</strong>
						</p>
						<div class="level-item level-is-shrinkable" v-if="scheduledOpen">
							<p>The queue is open until {{ closesAt }}.</p>
						</div>
						<p class="level-item level-is-shrinkable" v-else>
							The queue {{ opensAt }}.
						</p>
					</div>
				</div>
				<div class="block" v-else>
					<div style="margin-bottom: 0.5em">
						<b-skeleton></b-skeleton>
					</div>
				</div>
				<div class="buttons block" v-if="admin">
					<button
						class="button is-primary is-responsive"
						@click="editSchedule"
						v-if="queue.config !== null && queue.config.scheduled"
					>
						<span class="icon"><font-awesome-icon icon="calendar-alt"/></span>
						<span>Edit Schedule</span>
					</button>
					<button
						class="button is-warning is-responsive"
						@click="setOpen(false)"
						v-else-if="queue.open"
					>
						<span class="icon"><font-awesome-icon icon="lock"/></span>
						<span>Close Queue</span>
					</button>
					<button
						class="button is-success is-responsive"
						@click="setOpen(true)"
						v-else
					>
						<span class="icon"><font-awesome-icon icon="lock-open"/></span>
						<span>Open Queue</span>
					</button>
					<button class="button is-black is-responsive" @click="randomizeQueue">
						<span class="icon"><font-awesome-icon icon="dice"/></span>
						<span>Randomize Queue</span>
					</button>
					<button class="button is-danger is-responsive" @click="clearQueue">
						<span class="icon"><font-awesome-icon icon="eraser"/></span>
						<span>Clear Queue</span>
					</button>
					<button class="button is-light is-responsive" @click="broadcast">
						<span class="icon"><font-awesome-icon icon="bullhorn"/></span>
						<span>Broadcast to Queue</span>
					</button>
				</div>
				<div class="block">
					<h1 class="title">Sign Up</h1>
					<queue-signup :queue="queue" :time="time" />
				</div>
				<div class="block" v-if="admin && queue.stack.length > 0">
					<div class="level is-mobile">
						<div class="level-left">
							<div class="level-item">
								<p class="title">Stack</p>
							</div>
						</div>
						<div class="level-right">
							<div class="level-item">
								<button
									class="button is-small is-primary"
									@click="downloadStackAsCSV"
								>
									<span class="icon"><font-awesome-icon icon="download"/></span>
									<span>Download</span>
								</button>
							</div>
						</div>
					</div>
					<transition-group
						v-if="queue.stack.length > 0"
						name="entries-group"
						tag="div"
					>
						<div
							v-for="entry in queue.stack"
							:key="entry.id"
							class="block entries-group-item"
						>
							<queue-entry-display
								:entry="entry"
								:stack="true"
								:queue="queue"
								:admin="admin"
								:time="time"
							/>
						</div>
					</transition-group>
				</div>
			</div>
		</div>
	</div>
</template>

<script lang="ts">
import { Component, Prop, Vue } from 'vue-property-decorator';
import moment, { Moment } from 'moment-timezone';
import { json2csv } from 'json-2-csv';
import fileDownload from 'js-file-download';
import OrderedQueue from '@/types/OrderedQueue';
import { QueueEntry, RemovedQueueEntry } from '@/types/QueueEntry';
import QueueEntryDisplay from '@/components/ordered/QueueEntry.vue';
import QueueSignup from '@/components/ordered/QueueSignup.vue';
import ErrorDialog from '@/util/ErrorDialog';

import { library } from '@fortawesome/fontawesome-svg-core';
import {
	faStoreAltSlash,
	faGrinHearts,
	faHeartBroken,
	faUserGraduate,
	faEraser,
	faCalendarAlt,
	faDownload,
	faDice,
	faLock,
	faLockOpen,
} from '@fortawesome/free-solid-svg-icons';
import OrderedSchedule from './OrderedSchedule.vue';

library.add(
	faStoreAltSlash,
	faGrinHearts,
	faHeartBroken,
	faUserGraduate,
	faEraser,
	faCalendarAlt,
	faDownload,
	faDice,
	faLock,
	faLockOpen
);

@Component({
	components: {
		QueueSignup,
		QueueEntryDisplay,
		OrderedSchedule,
	},
})
export default class OrderedQueueDisplay extends Vue {
	@Prop({ required: true }) queue!: OrderedQueue;
	@Prop({ required: true }) loaded!: boolean;
	@Prop({ required: true }) ws!: WebSocket;
	@Prop({ required: true }) admin!: boolean;
	@Prop({ required: true }) time!: Moment;

	get open() {
		return this.queue.config?.scheduled ? this.scheduledOpen : this.queue.open;
	}

	get scheduledOpen() {
		return this.queue.scheduledOpen(this.time);
	}

	get closesAt() {
		return this.queue
			.halfHourToTime(
				this.queue.getNextCloseTime(this.queue.getHalfHour(this.time))
			)
			.format('LT');
	}

	get opensAt() {
		if (!this.queue.config?.scheduled) {
			return (
				'will be ' +
				(this.queue.open ? 'closed' : 'opened') +
				' manually by staff'
			);
		}

		const halfHour = this.queue.getNextOpenHalfHour(
			this.queue.getHalfHour(this.time)
		);

		if (halfHour === -1) {
			return 'is closed for the day';
		}

		return `opens at ${this.queue.halfHourToTime(halfHour).format('LT')}`;
	}

	clearQueue() {
		this.$buefy.dialog.confirm({
			title: 'Clear Queue',
			message: `Are you sure you want to clear the queue? <b>There's no undo; please don't use this to pop individual students.</b>`,
			type: 'is-danger',
			hasIcon: true,
			onConfirm: () => {
				fetch(process.env.BASE_URL + `api/queues/${this.queue.id}/entries`, {
					method: 'DELETE',
				}).then((res) => {
					if (res.status !== 204) {
						return ErrorDialog(res);
					}
				});
			},
		});
	}

	randomizeQueue() {
		this.$buefy.dialog.confirm({
			title: 'Randomize Queue',
			message: `Are you sure you want to randomize the queue? This will place everybody currently on the queue in a random position. This does not take into account the first-of-the-day status if your course uses it; consider whether this is okay. <b>There's no undo.</b>`,
			type: 'is-danger',
			hasIcon: true,
			onConfirm: () => {
				fetch(
					process.env.BASE_URL +
						`api/queues/${this.queue.id}/entries/randomize`,
					{
						method: 'POST',
					}
				).then((res) => {
					if (res.status !== 204) {
						return ErrorDialog(res);
					}
				});
			},
		});
	}

	editSchedule() {
		fetch(process.env.BASE_URL + `api/queues/${this.queue.id}/schedule`)
			.then((res) => res.json())
			.then((schedule) => {
				this.$buefy.modal.open({
					parent: this,
					component: OrderedSchedule,
					props: { defaultSchedule: schedule },
					events: {
						confirmed: (schedule: string[]) => {
							fetch(
								process.env.BASE_URL + `api/queues/${this.queue.id}/schedule`,
								{
									method: 'PUT',
									body: JSON.stringify(schedule),
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

	broadcast() {
		this.$buefy.dialog.prompt({
			message: `Broadcast message to all online users of queue`,
			trapFocus: true,
			onConfirm: (message) => {
				fetch(process.env.BASE_URL + `api/queues/${this.queue.id}/messages`, {
					method: 'POST',
					body: JSON.stringify({
						receiver: '<broadcast>',
						content: message,
					}),
				}).then((res) => {
					if (res.status !== 201) {
						return ErrorDialog(res);
					}
				});
			},
		});
	}

	downloadStackAsCSV() {
		fetch(process.env.BASE_URL + `api/queues/${this.queue.id}/stack`)
			.then((res) => res.json())
			.then((stack) => {
				json2csv(stack, (err, csv) => {
					if (err !== null || csv === undefined) {
						this.$buefy.dialog.alert({
							title: 'Error',
							message: `Failed to create csv: ${err}`,
							type: 'is-danger',
							hasIcon: true,
						});
						return;
					}
					fileDownload(csv, 'stack.csv', 'text/csv');
				});
			});
	}

	setOpen(open: boolean) {
		fetch(
			process.env.BASE_URL +
				`api/queues/${this.queue.id}/configuration/manual-open?` +
				new URLSearchParams({ open: open.toString() }),
			{
				method: 'PUT',
			}
		).then((res) => {
			if (res.status !== 204) {
				return ErrorDialog(res);
			}
		});
	}
}
</script>

<style scoped>
.entries {
	position: relative;
}

.entries-group-item {
	transition: all 0.8s ease;
}

.entries-group-enter-from {
	opacity: 0;
	transform: translateY(100%);
}

.entries-group-leave-to {
	opacity: 0;
	transform: translateY(-100%);
}

.entries-group-leave-active {
	position: absolute;
	width: 100%;
}

.level-is-shrinkable {
	flex-shrink: 1;
}
</style>
