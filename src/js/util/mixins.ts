import pull from 'lodash/pull';
import { assert, assertFalse } from './util';

export interface Message<Data_type = any> {
    category: string;
    data: Data_type;
    source: any;
}

// interface ObservableType {
//     send(category: string, data: any) : void;
//     addListener(listener: ObserverType, category?: string | string[]) : ObservableType;
//     removeListener(listener: ObserverType, category?: string) : ObservableType;
//     identify(category: string, func: (o:ObserverType) => any) : ObserverType;
// }

export function addListener(
    objWithObservable: { observable: Observable },
    listener: ObserverType,
    category?: string | string[],
) {
    objWithObservable.observable.addListener(listener, category);
}

export function messageResponse(messageCategory?: string) {
    return function (
        target: any,
        propertyKey: string,
        descriptor: PropertyDescriptor,
    ) {
        if (!target._act) {
            target._act = {};
        }
        target._act[messageCategory || propertyKey] = target[propertyKey];
    };
}

export interface MessageResponses {
    [index: string]: (msg: Message) => void;
}

export interface ObserverType {
    _act: MessageResponses;
}

// export class Observer {
//     private readonly actor: Actor;

//     constructor(actor: Actor) {
//         this.actor = actor;
//     }

//     public _IDENTIFY(msg : {data:(o:any) => void}) {
//         msg.data(this);
//     }

//     public listenTo(other: ObservableType, category: string) {
//         other.addListener(this, category);
//         return this;
//     }

//     public stopListeningTo(other: ObservableType, category: string) {
//         if (other) {
//             other.removeListener(this, category);
//         }
//         return this;
//     }

//     public recv (msg : Message) {

//         // Call the "_act" function for this
//         var catAct = this.actor._act[msg.category];
//         if (catAct){
//             catAct.call(this.actor, msg);
//         }
//         else if (this.actor._act._default) {
//             this.actor._act._default.call(this.actor, msg);
//         }
//         else {
//             assert(false);
//         }

//     }
// }

function receiveMessage(observer: ObserverType, msg: Message) {
    var catAct = observer._act[msg.category];
    if (catAct) {
        catAct.call(observer, msg);
    } else if (observer._act._default) {
        observer._act._default.call(observer, msg);
    } else {
        assertFalse();
    }
}

export class Observable {
    private silent = false;
    private universalObservers: ObserverType[] = [];
    private observers: { [index: string]: ObserverType[] } = {};

    private readonly source: any;

    constructor(source: any) {
        this.source = source;
    }

    public send(category: string, data?: any) {
        if (this.silent) {
            return;
        }

        let msg: Message = {
            category: category,
            data: data,
            source: this.source,
        };

        let observers = this.observers[msg.category];
        if (observers) {
            for (let i = 0; i < observers.length; ++i) {
                receiveMessage(observers[i], msg);
            }
        }

        for (let i = 0; i < this.universalObservers.length; ++i) {
            receiveMessage(this.universalObservers[i], msg);
        }
    }

    public addListener(listener: ObserverType, category?: string | string[]) {
        if (category) {
            if (Array.isArray(category)) {
                // If there's an array of categories, add to all individually
                for (var i = 0; i < category.length; ++i) {
                    this.addListener(listener, category[i]);
                }
            } else {
                if (!this.observers[category]) {
                    this.observers[category] = [];
                }
                this.observers[category].push(listener);
                this.listenerAdded(listener, category);
            }
        } else {
            // if no category, intent is to listen to everything
            this.universalObservers.push(listener);
            this.listenerAdded(listener);
        }
        return this;
    }

    /*
    Note: to remove a universal listener, you must call this with category==false.
    If a listener is universal, removing it from a particular category won't do anything.
    */
    public removeListener(listener: ObserverType, category?: string) {
        if (category) {
            // Remove from the list for a specific category (if list exists)
            let observers = this.observers[category];
            observers && pull(observers, listener);
            this.listenerRemoved(listener, category);
        } else {
            // Remove from all categories
            for (var cat in this.observers) {
                this.removeListener(listener, cat);
            }

            // Also remove from universal listeners
            pull(this.universalObservers, listener);
            this.listenerRemoved(listener);
        }
        return this;
    }

    protected listenerAdded(listener: ObserverType, category?: string): void {}
    protected listenerRemoved(
        listener: ObserverType,
        category?: string,
    ): void {}

    // public identify(category: string, func: (o:ObserverType) => any) {
    //     let other! : ObserverType; // Uses definite assignment annotation since the function is assumed to assign to other
    //     this.send(category, func || function(o:ObserverType) {other = o;});
    //     return other;
    // }
}
