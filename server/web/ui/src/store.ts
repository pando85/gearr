import { applyMiddleware, createStore, Store } from 'redux';
import { thunk } from 'redux-thunk';
import jobReducer, {JobState, initialState} from './jobReducer';
import {JobActionTypes} from './actions/JobActionsTypes';


const store: Store<JobState, JobActionTypes> = createStore(jobReducer, initialState, applyMiddleware(thunk));
export default store;

export type RootState = JobState;
export type AppDispatch = typeof store.dispatch
