module Test.Main where

import Prelude

import Data.Maybe (Maybe(..))
import Effect (Effect)
import Effect.Console (log)
import Effect.Exception (error, message)
import Effect.Ref as Ref
import Promise as Promise
import Promise.Internal as P
import Promise.Lazy as Lazy
import Promise.Rejection as Rejection
import Test.Assert as Assert

foreign import delay :: Int -> Effect (P.Promise Int)
foreign import failAfter :: Int -> Effect (P.Promise Int)

main :: Effect Unit
main = do
  log "Testing Promise.new, then_, catch, finally, all, race"

  -- Test resolve
  _ <- Promise.new (\res _ -> res "success")
    >>= Promise.then_
      ( \msg -> do
          Assert.assertEqual { actual: msg, expected: "success" }
          pure (Promise.resolve unit)
      )

  -- Test reject and catch
  _ <- Promise.new (\(_res :: Unit -> Effect Unit) rej -> rej (Rejection.fromError (error "fail")))
    >>= Promise.catch
      ( \err -> do
          case Rejection.toError err of
            Just e -> Assert.assert' "Should be fail" (message e == "fail")
            Nothing -> Assert.assert' "Expected an Error" false
          pure (Promise.resolve unit)
      )

  -- Test all
  _ <- Promise.all [ Promise.resolve 1, Promise.resolve 2, Promise.resolve 3 ]
    >>= Promise.then_
      ( \arr -> do
          Assert.assertEqual { actual: arr, expected: [ 1, 2, 3 ] }
          pure (Promise.resolve unit)
      )

  -- Test all with failure
  pFail <- failAfter 50
  pDelay <- delay 100
  _ <- Promise.all [ pDelay, pFail ]
    >>= Promise.then_
      ( \_ -> do
          Assert.assert' "Promise.all should have failed" false
          pure (Promise.resolve unit)
      )
    >>= Promise.catch (\_ -> pure (Promise.resolve unit))

  -- Test race
  p1 <- failAfter 500
  p2 <- delay 10 >>= Promise.then_ (\_ -> pure (Promise.resolve 42))
  _ <- Promise.race [ p1, p2 ]
    >>= Promise.then_
      ( \res -> do
          Assert.assertEqual { actual: res, expected: 42 }
          pure (Promise.resolve unit)
      )

  -- Test finally (success case)
  ref1 <- Ref.new 0
  _ <- Promise.new (\res _ -> res "ok")
    >>= Promise.finally (Ref.modify_ (_ + 1) ref1 >>= \_ -> pure (Promise.resolve unit))
    >>= Promise.then_
      ( \res -> do
          Assert.assertEqual { actual: res, expected: "ok" }
          val <- Ref.read ref1
          Assert.assertEqual { actual: val, expected: 1 }
          pure (Promise.resolve unit)
      )

  -- Test finally (failure case)
  ref2 <- Ref.new 0
  _ <- Promise.new (\(_res :: Unit -> Effect Unit) rej -> rej (Rejection.fromError (error "fail")))
    >>= Promise.finally (Ref.modify_ (_ + 1) ref2 >>= \_ -> pure (Promise.resolve unit))
    >>= Promise.catch
      ( \_ -> do
          val <- Ref.read ref2
          Assert.assertEqual { actual: val, expected: 1 }
          pure (Promise.resolve unit)
      )

  log "Testing Promise.Lazy"

  let
    lazyChain = do
      v1 <- Lazy.new (\res _ -> res 10)
      v2 <- pure 20
      pure (v1 + v2)

  _ <- Lazy.toPromise lazyChain
    >>= Promise.then_
      ( \res -> do
          Assert.assertEqual { actual: res, expected: 30 }
          pure (Promise.resolve unit)
      )

  let
    lazyFail = do
      v1 <- Lazy.fromPromise (failAfter 10)
      pure (v1 + 1)

  _ <- Lazy.toPromise (Lazy.catch (\_ -> pure 99) lazyFail)
    >>= Promise.then_
      ( \res -> do
          Assert.assertEqual { actual: res, expected: 99 }
          pure (Promise.resolve unit)
      )

  ref3 <- Ref.new 0
  let lazyFinally = Lazy.finally (Lazy.fromPromise (Ref.modify_ (_ + 1) ref3 >>= \_ -> pure (Promise.resolve unit))) (pure "ok")
  _ <- Lazy.toPromise lazyFinally
    >>= Promise.then_
      ( \res -> do
          Assert.assertEqual { actual: res, expected: "ok" }
          val <- Ref.read ref3
          Assert.assertEqual { actual: val, expected: 1 }
          pure (Promise.resolve unit)
      )

  -- Lazy All
  let lazyAll = Lazy.all [ pure 1, pure 2, pure 3 ]
  _ <- Lazy.toPromise lazyAll
    >>= Promise.then_
      ( \res -> do
          Assert.assertEqual { actual: res, expected: [ 1, 2, 3 ] }
          pure (Promise.resolve unit)
      )

  log "Done!"
