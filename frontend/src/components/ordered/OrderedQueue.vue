<template>
	<div class="columns">
		<div class="column is-6">
			<div class="entries">
				<h1 class="title">Queue</h1>
				<div v-if="loaded">
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
									<h2 class="subtitle block">See you next time!</h2>
								</span>
								<span v-else-if="admin">
									<font-awesome-icon
										icon="grin-hearts"
										size="10x"
										class="block"
									/>
									<h1 class="title block">The queue is empty.</h1>
									<h2 class="subtitle block">Good job! Yes, you!</h2>
								</span>
								<span v-else>
									<font-awesome-icon
										icon="heart-broken"
										size="10x"
										class="block"
									/>
									<h1 class="title block">The queue is empty.</h1>
									<h2 class="subtitle block">We're lonely over here!</h2>
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
			</div>
		</div>
		<div class="column is-5 is-offset-1">
			<div class="entries">
				<div class="level block" v-if="loaded">
					<div class="level-left">
						<p class="level-item">
							<font-awesome-icon icon="user-graduate" fixed-size />
							<strong>{{ queue.entries.length }}</strong>
						</p>
						<p class="level-item" v-if="open">
							The queue is open until {{ closesAt }}.
						</p>
						<p class="level-item" v-else>The queue {{ opensAt }}.</p>
					</div>
				</div>
				<div class="block" v-else>
					<div style="margin-bottom: 0.5em">
						<b-skeleton></b-skeleton>
					</div>
				</div>
				<div class="buttons block" v-if="admin">
					<button class="button is-danger" @click="clearQueue">
						Clear Queue
					</button>
				</div>
				<div class="block">
					<h1 class="title">Sign Up</h1>
					<queue-signup :queue="queue" :time="time" />
				</div>
				<div class="block" v-if="admin && queue.stack.length > 0">
					<h1 class="title">Stack</h1>
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
} from '@fortawesome/free-solid-svg-icons';

library.add(faStoreAltSlash, faGrinHearts, faHeartBroken, faUserGraduate);

@Component({
	components: {
		QueueSignup,
		QueueEntryDisplay,
	},
})
export default class OrderedQueueDisplay extends Vue {
	@Prop({ required: true }) queue!: OrderedQueue;
	@Prop({ required: true }) loaded!: boolean;
	@Prop({ required: true }) ws!: WebSocket;
	@Prop({ required: true }) admin!: boolean;
	@Prop({ required: true }) time!: Moment;

	get open() {
		return this.queue.open(this.time);
	}

	get closesAt() {
		return this.queue
			.halfHourToTime(
				this.queue.getNextCloseTime(this.queue.getHalfHour(this.time))
			)
			.format('LT');
	}

	get opensAt() {
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
</style>
