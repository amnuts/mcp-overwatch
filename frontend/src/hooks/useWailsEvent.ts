import { useEffect } from 'react';
import { Events } from '@wailsio/runtime';

export function useWailsEvent<T = unknown>(eventName: string, callback: (data: T) => void) {
    useEffect(() => {
        const cancel = Events.On(eventName, (event) => {
            callback(event.data as T);
        });
        return () => { cancel(); };
    }, [eventName, callback]);
}
