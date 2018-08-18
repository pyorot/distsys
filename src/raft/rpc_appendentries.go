package raft

// APPENDENTRIES RPC

// AppendEntriesArgs ...
type AppendEntriesArgs struct {
	Term         int
	LeaderID     int
	PrevLogIndex int
	PrevLogTerm  int
	Entries      []LogEntry
	LeaderCommit int
}

// AppendEntriesReply ...
type AppendEntriesReply struct {
	Term    int
	Success bool
}

// AppendEntries ...
func (rf *Raft) AppendEntries(args *AppendEntriesArgs, reply *AppendEntriesReply) {
	otherTerm := args.Term
	outcome, myTerm := rf.termSync(otherTerm, "AppendEntries", "receiver")
	react := outcome <= 0 // tS react: should I react to this RPC at all?
	reply.Term = myTerm

	// payload
	if react {
		rf.mu.Lock()
		// 1. set success
		reply.Success = len(rf.log) > args.PrevLogIndex && rf.log[args.PrevLogIndex].Term == args.PrevLogTerm
		// 2. merge log
		if reply.Success {
			rf.log = append(rf.log[:args.PrevLogIndex+1], args.Entries...)
		}

		/*  OLD
		// 2a. resolve conflicts
		for i := 0; i < len(args.Entries) && i < len(rf.log)-1-args.PrevLogIndex; i++ {
			if args.Entries[i].Term != rf.log[args.PrevLogIndex+1+i].Term {
				rf.log = rf.log[:args.PrevLogIndex+1+i]
				break
			}
		}
		// 2b. add payload
		rf.log = append(rf.log, args.Entries[len(rf.log)-(args.PrevLogIndex+1):]...)
		P("I:", rf.me, "; other:", args.LeaderID)
		P(rf.log)
		P(args.Entries)
		P(args.PrevLogIndex)
		*/

		// 3. update commitIndex
		newCommitIndex := Min(args.LeaderCommit, len(rf.log)-1)
		// P("!!", rf.me, rf.commitIndex, newCommitIndex, args.LeaderCommit, len(rf.log)-1)
		if newCommitIndex > rf.commitIndex {
			rf.commitIndex = newCommitIndex // has def increased
			go rf.applyEntries()
		}
		rf.mu.Unlock()
	}
	P("AppendEntries:", args.LeaderID, "<", rf.me, "|", otherTerm, "vs", myTerm, "| react", react, "| success", reply.Success)
}

// sendAppendEntries ...
func (rf *Raft) sendAppendEntries(server int, args *AppendEntriesArgs, reply *AppendEntriesReply) (ok bool) {
	P("AppendEntries:", rf.me, ">", server)
	ok = rf.peers[server].Call("Raft.AppendEntries", args, reply)
	// await reply here
	if !ok {
		P("AppendEntries:", rf.me, "?", server)
		return
	}
	otherTerm := reply.Term
	_, myTerm := rf.termSync(otherTerm, "AppendEntries", "sender")
	P("AppendEntries:", rf.me, "-", server, "|", myTerm, "vs", otherTerm)
	return
}
