module Concurrent

  # A new implementation of actor which also simulates the process, therefore it can be used
  # in the same way as Erlang's actors but **without** occupying thread. A tens of thousands
  # ProcessingActors can run at the same time sharing a thread pool.
  # @example
  #     # Runs on a pool, does not consume 50_000 threads
  #     actors = 50_000.times.map do |i|
  #       Concurrent::ProcessingActor.act(i) { |a, i| a.receive.then_on(:fast, i) { |m, i| m + i } }
  #     end
  #
  #     actors.each { |a| a.tell 1 }
  #     values = actors.map(&:termination).map(&:value)
  #     values[0,5]                                        # => [1, 2, 3, 4, 5]
  #     values[-5, 5]                                      # => [49996, 49997, 49998, 49999, 50000]
  # @!macro edge_warning
  class ProcessingActor < Synchronization::Object
    # TODO (pitr-ch 18-Dec-2016): (un)linking, bidirectional, sends special message, multiple link calls has no effect,
    # TODO (pitr-ch 21-Dec-2016): Make terminated a cancellation token?
    # link_spawn atomic, Can it be fixed by sending exit when linked dead actor?

    safe_initialization!

    # @return [Promises::Channel] actor's mailbox.
    def mailbox
      @Mailbox
    end

    # @return [Promises::Future(Object)] a future which is resolved when the actor ends its processing.
    #   It can either be fulfilled with a value when actor ends normally or rejected with
    #   a reason (exception) when actor fails.
    def termination
      @Terminated.with_hidden_resolvable
    end

    # Creates an actor.
    # @see #act_listening Behaves the same way, but does not take mailbox as a first argument.
    # @return [ProcessingActor]
    # @example
    #   actor = Concurrent::ProcessingActor.act do |actor|
    #     actor.receive.then do |message|
    #       # the actor ends normally with message
    #       message
    #     end
    #   end
    #
    #   actor.tell :a_message
    #       # => <#Concurrent::ProcessingActor:0x7fff11280560 termination:pending>
    #   actor.termination.value! # => :a_message
    def self.act(*args, &process)
      act_listening Promises::Channel.new, *args, &process
    end

    # Creates an actor listening to a specified channel (mailbox).
    # @param [Object] args Arguments passed to the process.
    # @param [Promises::Channel] channel which serves as mailing box. The channel can have limited
    #   size to achieve backpressure.
    # @yield args to the process to get back a future which represents the actors execution.
    # @yieldparam [Object] *args
    # @yieldreturn [Promises::Future(Object)] a future representing next step of execution
    # @return [ProcessingActor]
    # @example
    #   # TODO (pitr-ch 19-Jan-2017): actor with limited mailbox
    def self.act_listening(channel, *args, &process)
      actor = ProcessingActor.new channel
      Promises.
          future(actor, *args, &process).
          run.
          chain_resolvable(actor.instance_variable_get(:@Terminated))
      actor
    end

    # Receives a message when available, used in the actor's process.
    # @return [Promises::Future(Object)] a future which will be fulfilled with a message from
    #   mailbox when it is available.
    def receive(probe = Promises.resolvable_future)
      # TODO (pitr-ch 27-Dec-2016): patterns
      @Mailbox.pop probe
    end

    # Tells a message to the actor. May block current thread if the mailbox is full.
    # {#tell} is a better option since it does not block. It's usually used to integrate with
    # threading code.
    # @example
    #   Thread.new(actor) do |actor|
    #     # ...
    #     actor.tell! :a_message # blocks until the message is told
    #     #   (there is a space for it in the channel)
    #     # ...
    #   end
    # @param [Object] message
    # @return [self]
    def tell!(message)
      @Mailbox.push(message).wait!
      self
    end

    # Tells a message to the actor.
    # @param [Object] message
    # @return [Promises::Future(ProcessingActor)] a future which will be fulfilled with the actor
    #   when the message is pushed to mailbox.
    def tell(message)
      @Mailbox.push(message).then(self) { |_, actor| actor }
    end

    # Simplifies common pattern when a message sender also requires an answer to the message
    # from the actor. It appends a resolvable_future for the answer after the message.
    # @todo has to be nice also on the receive side, cannot make structure like this [message = [...], answer]
    #   all receives should receive something friendly
    # @param [Object] message
    # @param [Promises::ResolvableFuture] answer
    # @return [Promises::Future] a future which will be fulfilled with the answer to the message
    # @example
    #     add_once_actor = Concurrent::ProcessingActor.act do |actor|
    #       actor.receive.then do |(a, b), answer|
    #         result = a + b
    #         answer.fulfill result
    #         # terminate with result value
    #         result
    #       end
    #     end
    #     # => <#Concurrent::ProcessingActor:0x7fcd1315f6e8 termination:pending>
    #
    #     add_once_actor.ask([1, 2]).value!                  # => 3
    #     # fails the actor already added once
    #     add_once_actor.ask(%w(ab cd)).reason
    #     # => #<RuntimeError: actor terminated normally before answering with a value: 3>
    #     add_once_actor.termination.value!                  # => 3
    def ask(message, answer = Promises.resolvable_future)
      tell [message, answer]
      # do not leave answers unanswered when actor terminates.
      Promises.any(
          Promises.fulfilled_future(:answer).zip(answer),
          Promises.fulfilled_future(:termination).zip(@Terminated)
      ).chain do |fulfilled, (which, value), (_, reason)|
        # TODO (pitr-ch 20-Jan-2017): we have to know which future was resolved
        # TODO (pitr-ch 20-Jan-2017): make the combinator programmable, so anyone can create what is needed
        # TODO (pitr-ch 19-Jan-2017): ensure no callbacks are accumulated on @Terminated
        if which == :termination
          raise reason.nil? ? format('actor terminated normally before answering with a value: %s', value) : reason
        else
          fulfilled ? value : raise(reason)
        end
      end
    end

    # @return [String] string representation.
    def inspect
      format '<#%s:0x%x termination:%s>', self.class, object_id << 1, termination.state
    end

    private

    def initialize(channel = Promises::Channel.new)
      @Mailbox    = channel
      @Terminated = Promises.resolvable_future
      super()
    end

  end
end
