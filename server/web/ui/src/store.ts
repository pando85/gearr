import { applyMiddleware, createStore } from 'redux';
import { thunk } from 'redux-thunk';
import jobReducer, {JobState, initialState} from './jobReducer';


const store = createStore(jobReducer, initialState, applyMiddleware(thunk));
export default store;

export type RootState = JobState;
export type AppDispatch = typeof store.dispatch
